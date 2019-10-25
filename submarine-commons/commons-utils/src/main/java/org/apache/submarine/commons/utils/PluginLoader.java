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
package org.apache.submarine.commons.utils;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.File;
import java.io.IOException;
import java.lang.reflect.Constructor;
import java.lang.reflect.InvocationTargetException;
import java.lang.reflect.Method;
import java.net.URL;
import java.net.URLClassLoader;
import java.util.ArrayList;
import java.util.List;

public class PluginLoader<T> {
  private static final Logger LOG = LoggerFactory.getLogger(PluginLoader.class);

  public synchronized T loadPlugin(String pluginClassName)
      throws IOException {
    LOG.info("Loading Plug Class name: {}", pluginClassName);

    // load plugin from classpath directly first for these builtin Interpreter Plugin.
    T intpProcess = loadPluginFromClassPath(pluginClassName, null);
    if (intpProcess == null) {
      throw new IOException("Fail to load plugin: " + pluginClassName);
    }

    return intpProcess;
  }

  private T loadPluginFromClassPath(String pluginClassName, URLClassLoader pluginClassLoader) {
    T intpProcess = null;
    try {
      Class<?> clazz = null;
      if (null == pluginClassLoader) {
        clazz = Class.forName(pluginClassName);
      } else {
        clazz = Class.forName(pluginClassName, true, pluginClassLoader);
      }

      Constructor<?> cons[] = clazz.getConstructors();
      for (Constructor<?> constructor : cons) {
        LOG.debug(constructor.getName());
      }

      Method[] methods = clazz.getDeclaredMethods();
      //Loop through the methods and print out their names
      for (Method method : methods) {
        LOG.debug(method.getName());
      }

      intpProcess = (T) (clazz.getConstructor().newInstance());
      return intpProcess;
    } catch (InstantiationException | IllegalAccessException | ClassNotFoundException
        | NoSuchMethodException | InvocationTargetException e) {
      LOG.warn("Fail to instantiate InterpreterLauncher from classpath directly:", e);
    }

    return intpProcess;
  }

  // Get the class load from the specified path
  private URLClassLoader getPluginClassLoader(String pluginsDir, String pluginName)
      throws IOException {
    File pluginFolder = new File(pluginsDir + "/" + pluginName);
    if (!pluginFolder.exists() || pluginFolder.isFile()) {
      LOG.warn("PluginFolder {} doesn't exist or is not a directory", pluginFolder.getAbsolutePath());
      return null;
    }
    List<URL> urls = new ArrayList<>();
    for (File file : pluginFolder.listFiles()) {
      LOG.debug("Add file {} to classpath of plugin: {}", file.getAbsolutePath(), pluginName);
      urls.add(file.toURI().toURL());
    }
    if (urls.isEmpty()) {
      LOG.warn("Can not load plugin {}, because the plugin folder {} is empty.", pluginName, pluginFolder);
      return null;
    }
    return new URLClassLoader(urls.toArray(new URL[0]));
  }
}
