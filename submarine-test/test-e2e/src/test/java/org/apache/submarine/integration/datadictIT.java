/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package org.apache.submarine.integration;

import org.apache.submarine.AbstractSubmarineIT;
import org.apache.submarine.WebDriverManager;
import org.apache.submarine.SubmarineITUtils;
import org.openqa.selenium.support.ui.ExpectedConditions;
import org.openqa.selenium.support.ui.WebDriverWait;
import org.openqa.selenium.By;
import org.openqa.selenium.Keys;
import org.openqa.selenium.interactions.Actions;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.testng.annotations.AfterClass;
import org.testng.annotations.BeforeClass;
import org.testng.annotations.Test;
import org.testng.Assert;
import java.util.*;

public class datadictIT extends AbstractSubmarineIT {

  public final static Logger LOG = LoggerFactory.getLogger(datadictIT.class);

  @BeforeClass
  public static void startUp(){
    LOG.info("[Testcase]: datadictIT");
    driver =  WebDriverManager.getWebDriver();
  }

  @AfterClass
  public static void tearDown(){
    driver.quit();
  }

  @Test
  public void dataDictTest() throws Exception {
    // Login
    LOG.info("Login");
    pollingWait(By.cssSelector("input[ng-reflect-name='userName']"), MAX_BROWSER_TIMEOUT_SEC).sendKeys("admin");
    pollingWait(By.cssSelector("input[ng-reflect-name='password']"), MAX_BROWSER_TIMEOUT_SEC).sendKeys("admin");
    clickAndWait(By.cssSelector("button[class='login-form-button ant-btn ant-btn-primary']"));
    pollingWait(By.cssSelector("a[routerlink='/workbench/dashboard']"), MAX_BROWSER_TIMEOUT_SEC);

    // Start Routing & Navigation in data-dict 
    LOG.info("Start Routing & Navigation in data-dict");
    pollingWait(By.xpath("//span[contains(text(), \"Manager\")]"), MAX_BROWSER_TIMEOUT_SEC).click();
    WebDriverWait wait = new WebDriverWait( driver, 15, 5000);
    pollingWait(By.xpath("//a[@href='/workbench/manager/dataDict']"), MAX_BROWSER_TIMEOUT_SEC).click();
    wait.until(ExpectedConditions.visibilityOfElementLocated(By.xpath("//span[@class='ant-breadcrumb-link ng-star-inserted']")));
    Assert.assertEquals(driver.getCurrentUrl(), "http://localhost:8080/workbench/manager/dataDict");

    // Add button
    LOG.info("[TEST] Add button");
    // Add --> Ok --> required feedback
    pollingWait(By.cssSelector("form > nz-form-item:nth-child(3) > nz-form-control > div > span > button.ant-btn.ant-btn-default"), MAX_BROWSER_TIMEOUT_SEC).click();
    Assert.assertEquals( driver.findElements(By.xpath("//div[contains(text(), \"Add\")]")).size(), 1);
    pollingWait(By.cssSelector("button[class='ant-btn ng-star-inserted ant-btn-primary']"), MAX_BROWSER_TIMEOUT_SEC).click();
    Assert.assertEquals( driver.findElements(By.xpath("//div[contains(text(), \"Add\")]")).size(), 1);
    // Add --> Close
    pollingWait(By.cssSelector("button[class='ant-btn ng-star-inserted ant-btn-default']"), MAX_BROWSER_TIMEOUT_SEC).click();
    Assert.assertEquals( driver.findElements(By.xpath("//div[contains(text(), \"Add\")]")).size(), 0);
    // Add --> set input --> close
    pollingWait(By.cssSelector("form > nz-form-item:nth-child(3) > nz-form-control > div > span > button.ant-btn.ant-btn-default"), MAX_BROWSER_TIMEOUT_SEC).click(); 
    pollingWait(By.xpath("//input[@id='inputNewDictCode']"), MAX_BROWSER_TIMEOUT_SEC).sendKeys("test new dict code");
    pollingWait(By.xpath("//input[@id='inputNewDictName']"), MAX_BROWSER_TIMEOUT_SEC).sendKeys("test new dict name");
    pollingWait(By.xpath("//input[@id='inputNewDictDescription']"), MAX_BROWSER_TIMEOUT_SEC).sendKeys("test new dict description");
    pollingWait(By.cssSelector("button[class='ant-btn ng-star-inserted ant-btn-default']"), MAX_BROWSER_TIMEOUT_SEC).click();
    Assert.assertEquals( driver.findElements(By.xpath("//td[@id='dataDictCodetest new dict code']")).size(), 0);
    // Add --> set input --> ok --> new dict 
    pollingWait(By.cssSelector("form > nz-form-item:nth-child(3) > nz-form-control > div > span > button.ant-btn.ant-btn-default"), MAX_BROWSER_TIMEOUT_SEC).click(); 
    pollingWait(By.xpath("//input[@id='inputNewDictCode']"), MAX_BROWSER_TIMEOUT_SEC).sendKeys("test new dict code");
    pollingWait(By.xpath("//input[@id='inputNewDictName']"), MAX_BROWSER_TIMEOUT_SEC).sendKeys("test new dict name");
    pollingWait(By.xpath("//input[@id='inputNewDictDescription']"), MAX_BROWSER_TIMEOUT_SEC).sendKeys("test new dict description");
    pollingWait(By.cssSelector("button[class='ant-btn ng-star-inserted ant-btn-primary']"), MAX_BROWSER_TIMEOUT_SEC).click();
    Assert.assertEquals( driver.findElements(By.xpath("//td[@id='dataDictCodetest new dict code']")).size(), 1);

    // Configuration button
    LOG.info("[TEST] PROJECT_TYPE Configuration button");
    // old dict --> More --> Configuration --> Add --> set input --> OK
    pollingWait(By.xpath("//a[@id='dataDictMorePROJECT_TYPE']"), MAX_BROWSER_TIMEOUT_SEC).click();
    pollingWait(By.xpath("//li[@id='dataDictConfigurationPROJECT_TYPE']"), MAX_BROWSER_TIMEOUT_SEC).click();
    pollingWait(By.xpath("//button[@id='dataDictItemAddPROJECT_TYPE']"), MAX_BROWSER_TIMEOUT_SEC).click();
    pollingWait(By.xpath("//span[@class='ant-cascader-picker-label']"), MAX_BROWSER_TIMEOUT_SEC).click();
    pollingWait(By.xpath("//li[@title='unavailable']"), MAX_BROWSER_TIMEOUT_SEC).click();
    pollingWait(By.xpath("//input[@id='newItemCodePROJECT_TYPE']"), MAX_BROWSER_TIMEOUT_SEC).sendKeys("qqq");
    pollingWait(By.xpath("//input[@id='newItemNamePROJECT_TYPE']"), MAX_BROWSER_TIMEOUT_SEC).sendKeys("www");
    pollingWait(By.xpath("//button[@class='ant-btn ng-star-inserted ant-btn-default ant-btn-sm']"), MAX_BROWSER_TIMEOUT_SEC).click();
    Assert.assertEquals( driver.findElements(By.xpath("//td[contains(text(), \"qqq\")]")).size(), 1);
    pollingWait(By.xpath("//button[@class='ant-btn ng-star-inserted ant-btn-primary']"), MAX_BROWSER_TIMEOUT_SEC).click();
    // Check old dict
    pollingWait(By.xpath("//a[@id='dataDictMorePROJECT_TYPE']"), MAX_BROWSER_TIMEOUT_SEC).click();
    pollingWait(By.xpath("//li[@id='dataDictConfigurationPROJECT_TYPE']"), MAX_BROWSER_TIMEOUT_SEC).click();
    Assert.assertEquals( driver.findElements(By.xpath("//td[contains(text(), \"qqq\")]")).size(), 1);
    LOG.info("[TEST] End");
  }
}
