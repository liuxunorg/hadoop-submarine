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
package org.apache.submarine.server;

import org.apache.log4j.PropertyConfigurator;
import org.apache.submarine.server.rpc.SubmarineRpcServer;
import org.apache.submarine.server.workbench.websocket.NotebookServer;
import org.apache.submarine.commons.cluster.ClusterServer;
import org.eclipse.jetty.http.HttpVersion;
import org.eclipse.jetty.rewrite.handler.RewriteHandler;
import org.eclipse.jetty.rewrite.handler.RewriteRegexRule;
import org.eclipse.jetty.server.Handler;
import org.eclipse.jetty.server.HttpConfiguration;
import org.eclipse.jetty.server.HttpConnectionFactory;
import org.eclipse.jetty.server.SecureRequestCustomizer;
import org.eclipse.jetty.server.Server;
import org.eclipse.jetty.server.ServerConnector;
import org.eclipse.jetty.server.SslConnectionFactory;
import org.eclipse.jetty.server.handler.DefaultHandler;
import org.eclipse.jetty.server.handler.HandlerList;
import org.eclipse.jetty.server.session.SessionHandler;
import org.eclipse.jetty.servlet.DefaultServlet;
import org.eclipse.jetty.servlet.ServletContextHandler;
import org.eclipse.jetty.servlet.ServletHolder;
import org.eclipse.jetty.util.ssl.SslContextFactory;
import org.eclipse.jetty.util.thread.QueuedThreadPool;
import org.eclipse.jetty.util.thread.ThreadPool;
import org.eclipse.jetty.webapp.WebAppContext;
import org.eclipse.jetty.websocket.servlet.WebSocketServlet;
import org.glassfish.hk2.api.ServiceLocator;
import org.glassfish.hk2.api.ServiceLocatorFactory;
import org.glassfish.hk2.utilities.ServiceLocatorUtilities;
import org.glassfish.hk2.utilities.binding.AbstractBinder;
import org.glassfish.jersey.server.ResourceConfig;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import org.apache.submarine.commons.utils.SubmarineConfiguration;
import org.apache.submarine.commons.utils.SubmarineConfVars;

import javax.inject.Inject;
import javax.inject.Singleton;
import javax.servlet.DispatcherType;
import javax.servlet.RequestDispatcher;
import javax.servlet.ServletException;
import javax.servlet.http.HttpServlet;
import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;
import java.io.File;
import java.io.IOException;

public class SubmarineServer extends ResourceConfig {
  private static final Logger LOG = LoggerFactory.getLogger(SubmarineServer.class);

  private static long serverTimeStamp = System.currentTimeMillis();

  public static Server jettyWebServer;
  public static SubmarineRpcServer rpcServer;
  public static ServiceLocator sharedServiceLocator;

  private static SubmarineConfiguration conf = SubmarineConfiguration.getInstance();

  public static long getServerTimeStamp() {
    return serverTimeStamp;
  }

  public static void main(String[] args) throws InterruptedException,
      IOException {
    PropertyConfigurator.configure(ClassLoader.getSystemResource("log4j.properties"));

    final SubmarineConfiguration conf = SubmarineConfiguration.getInstance();
    LOG.info("Submarine server Host: " + conf.getServerAddress());
    if (conf.useSsl() == false) {
      LOG.info("Submarine server Port: " + conf.getServerPort());
    } else {
      LOG.info("Submarine server SSL Port: " + conf.getServerSslPort());
    }

    jettyWebServer = setupJettyServer(conf);

    // Web UI
    HandlerList handlers = new HandlerList();
    final WebAppContext webApp = setupWebAppContext(handlers, conf);
    jettyWebServer.setHandler(handlers);

    // Add
    sharedServiceLocator = ServiceLocatorFactory.getInstance().create("shared-locator");
    ServiceLocatorUtilities.enableImmediateScope(sharedServiceLocator);
    ServiceLocatorUtilities.bind(
        sharedServiceLocator,
        new AbstractBinder() {
          @Override
          protected void configure() {
            bindAsContract(NotebookServer.class)
                .to(WebSocketServlet.class)
                .in(Singleton.class);
          }
        });

    setupRestApiContextHandler(webApp, conf);

    // Notebook server
    setupNotebookServer(webApp, conf, sharedServiceLocator);

    // Cluster Server
    setupClusterServer();

    rpcServer = SubmarineRpcServer.startRpcServer();
    startServer();
  }

  @Inject
  public SubmarineServer() {
    packages("org.apache.submarine.server.workbench.rest",
             "org.apache.submarine.server.jobserver.rest",
             "org.apache.submarine.server.rest"
    );
  }

  private static void startServer() throws InterruptedException {
    LOG.info("Starting submarine server");
    try {
      jettyWebServer.start(); // Instantiates SubmarineServer
    } catch (Exception e) {
      LOG.error("Error while running jettyServer", e);
      System.exit(-1);
    }
    LOG.info("Done, submarine server started");

    Runtime.getRuntime()
        .addShutdownHook(
            new Thread(
                () -> {
                  LOG.info("Shutting down Submarine Server ... ");
                  try {
                    jettyWebServer.stop();
                    Thread.sleep(3000);
                  } catch (Exception e) {
                    LOG.error("Error while stopping servlet container", e);
                  }
                  LOG.info("Bye");
                }));

    jettyWebServer.join();
  }

  private static void setupRestApiContextHandler(WebAppContext webapp, SubmarineConfiguration conf) {
    final ServletHolder servletHolder =
        new ServletHolder(new org.glassfish.jersey.servlet.ServletContainer());

    servletHolder.setInitParameter("javax.ws.rs.Application", SubmarineServer.class.getName());
    servletHolder.setName("rest");
    servletHolder.setForcedPath("rest");
    webapp.setSessionHandler(new SessionHandler());
    webapp.addServlet(servletHolder, "/api/*");
  }

  private static WebAppContext setupWebAppContext(HandlerList handlers,
      SubmarineConfiguration conf) {
    WebAppContext webApp = new WebAppContext();
    webApp.setContextPath("/");
    File warPath = new File(conf.getString(SubmarineConfVars.ConfVars.WORKBENCH_WEB_WAR));
    LOG.info("workbench web war file path is {}.",
        conf.getString(SubmarineConfVars.ConfVars.WORKBENCH_WEB_WAR));
    if (warPath.isDirectory()) {
      // Development mode, read from FS
      // webApp.setDescriptor(warPath+"/WEB-INF/web.xml");
      webApp.setResourceBase(warPath.getPath());
      webApp.setParentLoaderPriority(true);
    } else {
      // use packaged WAR
      webApp.setWar(warPath.getAbsolutePath());
      File warTempDirectory = new File("webapps");
      warTempDirectory.mkdir();
      webApp.setTempDirectory(warTempDirectory);
    }
    // Explicit bind to root
    webApp.addServlet(new ServletHolder(new DefaultServlet()), "/*");

    // SUBMARINE-422.
    // Add error page mapping for context
    // webApp.addServlet(ErrorHandling.class, "/errorpage");
    // ErrorPageErrorHandler errorMapper = new ErrorPageErrorHandler();
    // errorMapper.addErrorPage(404, "/");
    // webApp.setErrorHandler(errorMapper);

    // to handle static resources against base resource (always last) always named "default" (per spec)
    // ServletHolder defaultHolder = new ServletHolder("default", DefaultServlet.class);
    // assigned to default url-pattern of "/" (per spec)
    // webApp.addServlet(defaultHolder, "/");

    RewriteHandler rewriteHandler = new RewriteHandler();
    rewriteHandler.setRewriteRequestURI(true);
    rewriteHandler.setRewritePathInfo(false);
    rewriteHandler.setOriginalPathAttribute("requestedPath");
    RewriteRegexRule rule = new RewriteRegexRule();
    rule.setRegex("/workbench/.*");
    rule.setReplacement("http://127.0.0.1:8080/");
    rewriteHandler.addRule(rule);

    handlers.setHandlers(new Handler[] { rewriteHandler, webApp, new DefaultHandler() });

    return webApp;
  }

  private static Server setupJettyServer(SubmarineConfiguration conf) {
    ThreadPool threadPool =
        new QueuedThreadPool(conf.getInt(SubmarineConfVars.ConfVars.SUBMARINE_SERVER_JETTY_THREAD_POOL_MAX),
            conf.getInt(SubmarineConfVars.ConfVars.SUBMARINE_SERVER_JETTY_THREAD_POOL_MIN),
            conf.getInt(SubmarineConfVars.ConfVars.SUBMARINE_SERVER_JETTY_THREAD_POOL_TIMEOUT));
    final Server server = new Server(threadPool);
    ServerConnector connector;

    if (conf.useSsl()) {
      LOG.debug("Enabling SSL for submarine Server on port " + conf.getServerSslPort());
      HttpConfiguration httpConfig = new HttpConfiguration();
      httpConfig.setSecureScheme("https");
      httpConfig.setSecurePort(conf.getServerSslPort());
      httpConfig.setOutputBufferSize(32768);
      httpConfig.setResponseHeaderSize(8192);
      httpConfig.setSendServerVersion(true);

      HttpConfiguration httpsConfig = new HttpConfiguration(httpConfig);
      SecureRequestCustomizer src = new SecureRequestCustomizer();
      httpsConfig.addCustomizer(src);

      connector = new ServerConnector(
              server,
              new SslConnectionFactory(getSslContextFactory(conf), HttpVersion.HTTP_1_1.asString()),
              new HttpConnectionFactory(httpsConfig));
    } else {
      connector = new ServerConnector(server);
    }

    configureRequestHeaderSize(conf, connector);
    // Set some timeout options to make debugging easier.
    int timeout = 1000 * 30;
    connector.setIdleTimeout(timeout);
    connector.setSoLingerTime(-1);
    connector.setHost(conf.getServerAddress());
    if (conf.useSsl()) {
      connector.setPort(conf.getServerSslPort());
    } else {
      connector.setPort(conf.getServerPort());
    }

    server.addConnector(connector);
    return server;
  }

  private static void setupNotebookServer(WebAppContext webapp,
      SubmarineConfiguration conf, ServiceLocator serviceLocator) {
    String maxTextMessageSize = conf.getWebsocketMaxTextMessageSize();
    final ServletHolder servletHolder =
        new ServletHolder(serviceLocator.getService(NotebookServer.class));
    servletHolder.setInitParameter("maxTextMessageSize", maxTextMessageSize);

    final ServletContextHandler context = new ServletContextHandler(ServletContextHandler.SESSIONS);
    webapp.addServlet(servletHolder, "/ws/*");
  }

  private static void setupClusterServer() {
    if (conf.isClusterMode()) {
      ClusterServer clusterServer = ClusterServer.getInstance();
      clusterServer.start();
    }
  }

  private static SslContextFactory getSslContextFactory(SubmarineConfiguration conf) {
    SslContextFactory sslContextFactory = new SslContextFactory();

    // Set keystore
    sslContextFactory.setKeyStorePath(conf.getKeyStorePath());
    sslContextFactory.setKeyStoreType(conf.getKeyStoreType());
    sslContextFactory.setKeyStorePassword(conf.getKeyStorePassword());
    sslContextFactory.setKeyManagerPassword(conf.getKeyManagerPassword());

    if (conf.useClientAuth()) {
      sslContextFactory.setNeedClientAuth(conf.useClientAuth());

      // Set truststore
      sslContextFactory.setTrustStorePath(conf.getTrustStorePath());
      sslContextFactory.setTrustStoreType(conf.getTrustStoreType());
      sslContextFactory.setTrustStorePassword(conf.getTrustStorePassword());
    }

    return sslContextFactory;
  }

  private static void configureRequestHeaderSize(
      SubmarineConfiguration conf, ServerConnector connector) {
    HttpConnectionFactory cf =
        (HttpConnectionFactory) connector.getConnectionFactory(HttpVersion.HTTP_1_1.toString());
    int requestHeaderSize = conf.getJettyRequestHeaderSize();
    cf.getHttpConfiguration().setRequestHeaderSize(requestHeaderSize);
  }

  public static class ErrorHandling extends HttpServlet {
    @Override
    protected void doGet(HttpServletRequest req, HttpServletResponse resp)
      throws ServletException, IOException {
      if (req.getDispatcherType() != DispatcherType.ERROR) {
        // we didn't get here from a error dispatch.
        // somebody attempted to use this servlet directly.
        resp.setStatus(404);
        return;
      }

      String requestedResource = (String) req.getAttribute(RequestDispatcher.ERROR_REQUEST_URI);
      LOG.error("[ErrorHandling] Requested resource was " + requestedResource);
      int statusCode = (int) req.getAttribute(RequestDispatcher.ERROR_STATUS_CODE);
      switch (statusCode) {
        case 404:
          // let handle it by a redirect
          // resp.sendRedirect("/");
          break;
        default:
          // pass the other errors through
          resp.setStatus(statusCode);
          break;
      }
    }
  }
}
