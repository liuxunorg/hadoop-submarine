/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */
package org.apache.submarine.launcher.docker;

import com.google.common.annotations.VisibleForTesting;
import com.google.common.base.Charsets;
import com.google.common.io.Files;
import com.google.common.io.Resources;

import java.io.File;
import java.io.FileInputStream;
import java.io.IOException;
import java.io.InputStream;
import java.net.URI;
import java.net.URL;
import java.util.ArrayList;
import java.util.Date;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Properties;
import java.util.TimeZone;
import java.util.concurrent.atomic.AtomicBoolean;

import com.spotify.docker.client.DefaultDockerClient;
import com.spotify.docker.client.DockerClient;
import com.spotify.docker.client.LogStream;
import com.spotify.docker.client.ProgressHandler;
import com.spotify.docker.client.exceptions.DockerException;
import com.spotify.docker.client.messages.Container;
import com.spotify.docker.client.messages.ContainerConfig;
import com.spotify.docker.client.messages.ContainerCreation;
import com.spotify.docker.client.messages.ExecCreation;
import com.spotify.docker.client.messages.HostConfig;
import com.spotify.docker.client.messages.PortBinding;
import com.spotify.docker.client.messages.ProgressMessage;
import org.apache.commons.lang.StringUtils;
import org.apache.submarine.commons.utils.JinjiaTemplate;
import org.apache.submarine.commons.utils.NetworkUtils;
import org.apache.submarine.commons.utils.SubmarineConfiguration;
import org.apache.submarine.commons.utils.tar.TarFileEntry;
import org.apache.submarine.commons.utils.tar.TarUtils;
import org.apache.submarine.launcher.LauncherProcess;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import static java.nio.charset.StandardCharsets.UTF_8;

public class LauncherDocker extends LauncherProcess {
  private static final Logger LOG = LoggerFactory.getLogger(LauncherDocker.class);

  private final Map<String, String> envs;

  private String dockerIntpServicePort = "0";

  private AtomicBoolean dockerStarted = new AtomicBoolean(false);

  private DockerClient docker = null;
  private final String containerName;
  private static final String WORKBENCH_DAEMON_JINJA = "/jinja_templates/workbench-daemon.jinja";

  private SubmarineConfiguration sconf = SubmarineConfiguration.create();

  private String submarineHome;

  @VisibleForTesting
  final String DOCKER_HOST;

  private String containerId;
  final String CONTAINER_UPLOAD_TAR_DIR = "/tmp/submarine-tar";

  public LauncherDocker() {
    this.envs = new HashMap();
    this.containerName = this.launcherId;

    try {
      this.submarineHome = sconf.getSubmarineHome();
    } catch (IOException e) {
      LOG.error(e.getMessage(), e);
    }
    String defDockerHost = "http://0.0.0.0:2375";
    String dockerHost = System.getenv("DOCKER_HOST");
    DOCKER_HOST = (dockerHost == null) ?  defDockerHost : dockerHost;
  }

  @Override
  public void start() throws IOException {
    docker = DefaultDockerClient.builder().uri(URI.create(DOCKER_HOST)).build();

    removeExistContainer(containerName);

    final Map<String, List<PortBinding>> portBindings = new HashMap<>();

    // Bind container ports to host ports
    int intpServicePort = NetworkUtils.findRandomAvailablePortOnAllLocalInterfaces();
    this.dockerIntpServicePort = String.valueOf(intpServicePort);
    final String[] ports = {dockerIntpServicePort};
    for (String port : ports) {
      List<PortBinding> hostPorts = new ArrayList<>();
      hostPorts.add(PortBinding.of("0.0.0.0", port));
      portBindings.put(port, hostPorts);
    }

    final HostConfig hostConfig = HostConfig.builder()
        .networkMode("host").portBindings(portBindings).build();

    JinjiaTemplate specTemplate = new JinjiaTemplate();
    specTemplate.loadProperties(getTemplateBindings());
    URL urlTemplate = this.getClass().getResource(WORKBENCH_DAEMON_JINJA);
    String template = Resources.toString(urlTemplate, Charsets.UTF_8);
    String dockerCommand = specTemplate.render(template);
    int firstLineIsNewline = dockerCommand.indexOf("\n");
    if (firstLineIsNewline == 0) {
      dockerCommand = dockerCommand.replaceFirst("\n", "");
    }
    LOG.info("dockerCommand = {}", dockerCommand);

    List<String> listEnv = getListEnvs();
    LOG.info("docker listEnv = {}", listEnv);

    // check if the container process exit script
    // if process exit, then container need exit
    StringBuilder sbStartCmd = new StringBuilder();
    sbStartCmd.append("sleep 10; ");
    sbStartCmd.append("process=" + getSuperClassName() + "; ");
    sbStartCmd.append("RUNNING_PIDS=$(ps x | grep $process | grep -v grep | awk '{print $1}'); ");
    sbStartCmd.append("while [ ! -z \"$RUNNING_PIDS\" ]; ");
    sbStartCmd.append("do sleep 1; ");
    sbStartCmd.append("RUNNING_PIDS=$(ps x | grep $process | grep -v grep | awk '{print $1}'); ");
    sbStartCmd.append("done");

    String hostAddr = NetworkUtils.findAvailableHostAddress();

    // Create container with exposed ports
    final ContainerConfig containerConfig = ContainerConfig.builder()
        .hostConfig(hostConfig)
        .hostname(hostAddr)
        .image(containerImage)
        .workingDir("/")
        .env(listEnv)
        .cmd("sh", "-c", sbStartCmd.toString())
        .build();

    try {
      LOG.info("wait docker pull image {} ...", containerImage);
      docker.pull(containerImage, new ProgressHandler() {
        @Override
        public void progress(ProgressMessage message) throws DockerException {
          if (null != message.error()) {
            LOG.error(message.toString());
          }
        }
      });

      final ContainerCreation containerCreation
          = docker.createContainer(containerConfig, containerName);
      this.containerId = containerCreation.id();

      // Start container
      docker.startContainer(containerId);

      copyConfigureFileToContainer(containerId);

      execInContainer(containerId, dockerCommand, false);
    } catch (DockerException | InterruptedException e) {
      LOG.error(e.getMessage(), e);
      throw new IOException(e.getMessage());
    }

    long startTime = System.currentTimeMillis();

    long timeout = sconf.getLauncherTimeout();
    // wait until interpreter send dockerStarted message through thrift rpc
    synchronized (dockerStarted) {
      if (!dockerStarted.get()) {
        try {
          dockerStarted.wait(timeout);
        } catch (InterruptedException e) {
          LOG.error("Remote interpreter is not accessible");
          throw new IOException(e.getMessage());
        }
      }
    }

    if (!dockerStarted.get()) {
      LOG.info("Interpreter docker creation is time out in {} seconds", timeout / 1000);
    }

  }

  @VisibleForTesting
  Properties getTemplateBindings() throws IOException {
    Properties dockerProperties = new Properties();

    // docker template properties
    dockerProperties.put("CONTAINER_SUBMARINE_HOME", submarineHome);

    return dockerProperties;
  }

  @VisibleForTesting
  List<String> getListEnvs() {
    // environment variables
    envs.put("SUBMARINE_HOME", submarineHome);
    envs.put("SUBMARINE_CONF_DIR", submarineHome + "/conf");

    // set container time zone
    String dockerTimeZone = System.getenv("DOCKER_TIME_ZONE");
    if (StringUtils.isBlank(dockerTimeZone)) {
      dockerTimeZone = TimeZone.getDefault().getID();
    }
    envs.put("TZ", dockerTimeZone);

    List<String> listEnv = new ArrayList<>();
    for (Map.Entry<String, String> entry : this.envs.entrySet()) {
      String env = entry.getKey() + "=" + entry.getValue();
      listEnv.add(env);
    }
    return listEnv;
  }

  @Override
  public void stop() {
    try {
      // Kill container
      docker.killContainer(containerName);

      // Remove container
      docker.removeContainer(containerName);
    } catch (DockerException | InterruptedException e) {
      LOG.error(e.getMessage(), e);
    }

    // Close the docker client
    docker.close();
  }

  // Because docker can't create a container with the same name, it will cause the creation to fail.
  // If the submarine service is abnormal and the container that was created is not closed properly,
  // the container will not be created again.
  private void removeExistContainer(String containerName) {
    boolean isExist = false;
    try {
      final List<Container> containers
          = docker.listContainers(DockerClient.ListContainersParam.allContainers());
      for (Container container : containers) {
        for (String name : container.names()) {
          // because container name like '/md-shared', so need add '/'
          if (StringUtils.equals(name, "/" + containerName)) {
            isExist = true;
            break;
          }
        }
      }

      if (isExist == true) {
        LOG.info("kill exist container {}", containerName);
        docker.killContainer(containerName);
      }
    } catch (DockerException | InterruptedException e) {
      LOG.error(e.getMessage(), e);
    } finally {
      try {
        if (isExist == true) {
          docker.removeContainer(containerName);
        }
      } catch (DockerException | InterruptedException e) {
        LOG.error(e.getMessage(), e);
      }
    }
  }

  // upload configure file to submarine interpreter container
  // keytab file & submarine-site.xml & krb5.conf
  // The submarine configures the mount file into the container through `localization`
  // NOTE: The path to the file uploaded to the container,
  // Can not be repeated, otherwise it will lead to failure.
  private void copyConfigureFileToContainer(String containerId)
      throws IOException, DockerException, InterruptedException {
    HashMap<String, String> copyFiles = new HashMap<>();

    // Rebuild directory
    rmInContainer(containerId, submarineHome);
    mkdirInContainer(containerId, submarineHome);

    // 1) submarine-site.xml is uploaded to `${CONTAINER_SUBMARINE_HOME}` directory in the container
    String confPath = "/conf";
    String sconfPath = getPathByHome(submarineHome, confPath);
    mkdirInContainer(containerId, sconfPath);
    copyFiles.put(sconfPath + "/submarine-site.xml", sconfPath + "/submarine-site.xml");
    copyFiles.put(sconfPath + "/log4j.properties", sconfPath + "/log4j.properties");

    // 2) upload krb5.conf to container
    String krb5conf = "/etc/krb5.conf";
    File krb5File = new File(krb5conf);
    if (krb5File.exists()) {
      rmInContainer(containerId, krb5conf);
      copyFiles.put(krb5conf, krb5conf);
    } else {
      LOG.warn("{} file not found, Did not upload the krb5.conf to the container!", krb5conf);
    }

    deployToContainer(containerId, copyFiles);
  }

  private void deployToContainer(String containerId, HashMap<String, String> copyFiles)
      throws InterruptedException, DockerException, IOException {
    // mkdir CONTAINER_UPLOAD_TAR_DIR
    mkdirInContainer(containerId, CONTAINER_UPLOAD_TAR_DIR);

    // file tar package
    String tarFile = file2Tar(copyFiles);

    // copy tar to submarine_CONTAINER_DIR, auto unzip
    InputStream inputStream = new FileInputStream(tarFile);
    try {
      docker.copyToContainer(inputStream, containerId, CONTAINER_UPLOAD_TAR_DIR);
    } finally {
      inputStream.close();
    }

    // copy all files in CONTAINER_UPLOAD_TAR_DIR to the root directory
    cpInContainer(containerId, CONTAINER_UPLOAD_TAR_DIR + "/*", "/");

    // delete tar file in the local
    File fileTar = new File(tarFile);
    fileTar.delete();
  }

  private void mkdirInContainer(String containerId, String path)
      throws DockerException, InterruptedException {
    String execCommand = "mkdir " + path + " -p";
    execInContainer(containerId, execCommand, true);
  }

  private void rmInContainer(String containerId, String path)
      throws DockerException, InterruptedException {
    String execCommand = "rm " + path + " -R";
    execInContainer(containerId, execCommand, true);
  }

  private void cpInContainer(String containerId, String from, String to)
      throws DockerException, InterruptedException {
    String execCommand = "cp " + from + " " + to + " -R";
    execInContainer(containerId, execCommand, true);
  }

  private void execInContainer(String containerId, String execCommand, boolean logout)
      throws DockerException, InterruptedException {

    LOG.info("exec container commmand: " + execCommand);

    final String[] command = {"sh", "-c", execCommand};
    final ExecCreation execCreation = docker.execCreate(
        containerId, command, DockerClient.ExecCreateParam.attachStdout(),
        DockerClient.ExecCreateParam.attachStderr());

    LogStream logStream = docker.execStart(execCreation.id());
    while (logStream.hasNext() && logout) {
      final String log = UTF_8.decode(logStream.next().content()).toString();
      LOG.info(log);
    }
  }

  private String file2Tar(HashMap<String, String> copyFiles) throws IOException {
    File tmpDir = Files.createTempDir();

    Date date = new Date();
    String tarFileName = tmpDir.getPath() + date.getTime() + ".tar";

    List<TarFileEntry> tarFileEntries = new ArrayList<>();
    for (Map.Entry<String, String> entry : copyFiles.entrySet()) {
      String filePath = entry.getKey();
      String archivePath = entry.getValue();
      TarFileEntry tarFileEntry = new TarFileEntry(new File(filePath), archivePath);
      tarFileEntries.add(tarFileEntry);
    }

    TarUtils.compress(tarFileName, tarFileEntries);

    return tarFileName;
  }

  // ${SUBMARINE_HOME}/lib/interpreter
  private String getPathByHome(String homeDir, String path) throws IOException {
    File file = null;
    if (null == homeDir || StringUtils.isEmpty(homeDir)) {
      file = new File(path);
    } else {
      file = new File(homeDir, path);
    }
    if (file.exists()) {
      return file.getAbsolutePath();
    }

    throw new IOException("Can't find directory in " + homeDir + path + "!");
  }
}
