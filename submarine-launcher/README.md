<!--
   Licensed to the Apache Software Foundation (ASF) under one or more
   contributor license agreements.  See the NOTICE file distributed with
   this work for additional information regarding copyright ownership.
   The ASF licenses this file to You under the Apache License, Version 2.0
   (the "License"); you may not use this file except in compliance with
   the License.  You may obtain a copy of the License at
   http://www.apache.org/licenses/LICENSE-2.0
   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
-->
# Launcher

Submarine Workbench provides a separate Docker container for each user, which allows each user to use their own interpreter, set up github tonken, write code, etc. in their completely separate runtime environment.
Submarine Workbench provides three ways of running YARN/K8s/Docker docker, allowing users to use it in different runtime environments.
The Launcher component is designed to be designed to support both of these modes of operation, running the user's docker container in a different resource scheduling system based on the settings in Submarine Configure.


## Launcer Docker
Submarine Workbench Server creates a user's Workspace container in the native Docker environment by using launcherDocker. In addition, Workbench Server integrates Submarine Cluster Server to enable multiple Submarine Workbench Server groups to be built into Submarine Workbench Server Cluster, providing a scaleable Workspace container runtime environment.
