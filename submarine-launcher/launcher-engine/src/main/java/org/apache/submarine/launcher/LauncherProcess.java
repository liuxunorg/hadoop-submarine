/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */
package org.apache.submarine.launcher;

import org.apache.commons.lang.StringUtils;
import org.apache.submarine.commons.utils.PluginLoader;
import org.apache.submarine.commons.utils.SubmarineConfiguration;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;

/**
 * Entry point for Submarine Launcher process.
 */
public class LauncherProcess extends LauncherIFace {
  private static final Logger LOG = LoggerFactory.getLogger(LauncherProcess.class);

  private static SubmarineConfiguration sconf = SubmarineConfiguration.create();

  private LauncherProcess launcherProcess;

  protected LauncherMode launcherMode;
  protected LauncherType launcherType;
  protected String launcherId;
  protected String containerImage;

  public static void main(String[] args) throws IOException {
    String type = args[0];
    String id = args[1];
    String containerImage = "";
    LauncherType launcherType = LauncherType.UNKONW_TYPE;

    if (StringUtils.equals(type, "SUBMARINE_SERVER")) {
      launcherType = LauncherType.SUBMARINE_SERVER;
      containerImage = sconf.getSubmarineServerImage();
    } else if (StringUtils.equals(type, "WORKBENCH_SERVER")) {
      launcherType = LauncherType.WORKBENCH_SERVER;
      containerImage = sconf.getWorkbenchServerImage();
    } else if (StringUtils.equals(type, "USER_WORKSPACE")) {
      launcherType = LauncherType.USER_WORKSPACE;
      containerImage = sconf.getUserWorkspaceImage();
    } else {
      LOG.error("Unsupported launcher type:{}", type);
    }

    LauncherProcess launcherProcess = new LauncherProcess(launcherType, id, containerImage);
    int launcherTimeout = sconf.getLauncherTimeout();
    launcherProcess.start();
  }

  public LauncherProcess() {}

  public LauncherProcess(LauncherType launcherType, String id, String containerImage)
      throws IOException {
    this.launcherType = launcherType;
    this.launcherId = id;
    this.containerImage = containerImage;

    PluginLoader<LauncherProcess> pluginLoader = new PluginLoader();
    String superClassName = getSuperClassName(launcherMode);
    this.launcherProcess = pluginLoader.loadPlugin(superClassName);
  }

  // get super launcher class name
  private static String getSuperClassName(LauncherMode launcherMode) {
    String superLauncherClassName = "";

    switch (launcherMode) {
      case LOCAL:
        superLauncherClassName = "org.apache.submarine.launcher.LauncherProcess";
        break;
      case DOCKER:
        superLauncherClassName = "org.apache.submarine.launcher.LauncherDocker";
        break;
      default:
        LOG.error("Unsupported launcher mode:{}", launcherMode);
        break;
    }

    return superLauncherClassName;
  }

  protected String getSuperClassName() {
    return this.getClass().getName();
  }

  @Override
  public void start() throws IOException {
    LOG.error("Please implement the start() method of the child class!");
  }

  @Override
  public void stop() throws IOException {
    LOG.error("Please implement the stop() method of the child class!");
  }

  enum LauncherMode {
    LOCAL, //
    DOCKER,
    K8s,
    YARN
  }

  enum LauncherType {
    UNKONW_TYPE,
    SUBMARINE_SERVER,
    WORKBENCH_SERVER,
    USER_WORKSPACE
  }
}
