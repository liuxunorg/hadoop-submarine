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

import { CommonModule } from '@angular/common';
import { NgModule } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ComponentsModule } from '@submarine/components/components.module';
import { NgZorroAntdModule } from 'ng-zorro-antd';
import { NewProjectPageComponent } from './project/new-project-page/new-project-page.component';
import { ProjectComponent } from './project/project.component';
import { ReleaseComponent } from './release/release.component';
import { SharedComponent } from './shared/shared.component';
import { TeamComponent } from './team/team.component';
import { TrainingComponent } from './training/training.component';

@NgModule({
    declarations: [
      ProjectComponent,
      ReleaseComponent,
      TrainingComponent,
      TeamComponent,
      SharedComponent,
      NewProjectPageComponent
    ],
    imports: [
      CommonModule,
      ComponentsModule,
      NgZorroAntdModule,
      FormsModule
    ],
    exports: [
      ProjectComponent,
      ReleaseComponent,
      TrainingComponent,
      TeamComponent,
      SharedComponent,
      NewProjectPageComponent
    ]
  })
  export class WorkspaceModule {
  }
