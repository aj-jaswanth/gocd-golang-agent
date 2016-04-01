/*
 * Copyright 2016 ThoughtWorks, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *  http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package agent_test

import (
	"github.com/bmatcuk/doublestar"
	. "github.com/gocd-contrib/gocd-golang-agent/agent"
	"github.com/gocd-contrib/gocd-golang-agent/protocol"
	"github.com/xli/assert"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestExport(t *testing.T) {
	setUp(t)
	defer tearDown()

	os.Setenv("TEST_EXPORT", "EXPORT_VALUE")
	defer os.Setenv("TEST_EXPORT", "")

	goServer.SendBuild(AgentId, buildId,
		protocol.ExportCommand("env1", "value1", "false"),
		protocol.ExportCommand("env2", "value2", "true"),
		protocol.ExportCommand("env1", "value4", "false"),
		protocol.ExportCommand("env2", "value5", "true"),
		protocol.ExportCommand("env2", "value6", "false"),
		protocol.ExportCommand("env2", "value6", ""),
		protocol.ExportCommand("env2", "", ""),
		protocol.ExportCommand("TEST_EXPORT"),
	)
	assert.Equal(t, "agent Building", stateLog.Next())
	assert.Equal(t, "build Passed", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())

	log, err := goServer.ConsoleLog(buildId)
	assert.Nil(t, err)
	expected := `setting environment variable 'env1' to value 'value1'
setting environment variable 'env2' to value '********'
overriding environment variable 'env1' with value 'value4'
overriding environment variable 'env2' with value '********'
overriding environment variable 'env2' with value 'value6'
overriding environment variable 'env2' with value 'value6'
overriding environment variable 'env2' with value ''
setting environment variable 'TEST_EXPORT' to value 'EXPORT_VALUE'
`
	assert.Equal(t, expected, trimTimestamp(log))
}

func TestMkdirCommand(t *testing.T) {
	setUp(t)
	defer tearDown()

	wd := pipelineDir()
	goServer.SendBuild(AgentId, buildId,
		protocol.MkdirsCommand("path/in/pipeline/dir").Setwd(relativePath(wd)),
	)
	assert.Equal(t, "agent Building", stateLog.Next())
	assert.Equal(t, "build Passed", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())
	_, err := os.Stat(filepath.Join(wd, "path/in/pipeline/dir"))
	assert.Nil(t, err)
}

func TestCleandirCommand(t *testing.T) {
	setUp(t)
	defer tearDown()

	wd := createTestProjectInPipelineDir()
	goServer.SendBuild(AgentId, buildId,
		protocol.CleandirCommand("test", "world2").Setwd(relativePath(wd)),
	)
	assert.Equal(t, "agent Building", stateLog.Next())
	assert.Equal(t, "build Passed", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())

	matches, err := doublestar.Glob(filepath.Join(wd, "**/*.txt"))
	assert.Nil(t, err)
	sort.Strings(matches)
	expected := []string{
		"0.txt",
		"src/1.txt",
		"src/2.txt",
		"src/hello/3.txt",
		"src/hello/4.txt",
		"test/world2/10.txt",
		"test/world2/11.txt",
	}

	for i, f := range matches {
		actual := f[len(wd)+1:]
		assert.Equal(t, expected[i], actual)
	}
}

func TestFailCommand(t *testing.T) {
	setUp(t)
	defer tearDown()

	goServer.SendBuild(AgentId, buildId, protocol.FailCommand("something is wrong, please fail"))
	assert.Equal(t, "agent Building", stateLog.Next())
	assert.Equal(t, "build Failed", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())

	log, err := goServer.ConsoleLog(buildId)
	assert.Nil(t, err)
	expected := Sprintf("ERROR: something is wrong, please fail\n")
	assert.Equal(t, expected, trimTimestamp(log))
}

func TestSecretCommand(t *testing.T) {
	setUp(t)
	defer tearDown()

	goServer.SendBuild(AgentId, buildId,
		protocol.SecretCommand("thisissecret", "$$$$$$"),
		protocol.SecretCommand("replacebydefaultmask"),
		protocol.EchoCommand("hello (thisissecret)"),
		protocol.EchoCommand("hello (replacebydefaultmask)"),
	)
	assert.Equal(t, "agent Building", stateLog.Next())
	assert.Equal(t, "build Passed", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())

	log, err := goServer.ConsoleLog(buildId)
	assert.Nil(t, err)
	expected := Sprintf("hello ($$$$$$)\nhello (********)\n")
	assert.Equal(t, expected, trimTimestamp(log))
}

func TestShouldMaskSecretInExecOutput(t *testing.T) {
	setUp(t)
	defer tearDown()

	goServer.SendBuild(AgentId, buildId,
		protocol.SecretCommand("thisissecret", "$$$$$$"),
		protocol.ExecCommand("echo", "hello (thisissecret)"),
	)
	assert.Equal(t, "agent Building", stateLog.Next())
	assert.Equal(t, "build Passed", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())

	log, err := goServer.ConsoleLog(buildId)
	assert.Nil(t, err)
	expected := Sprintf("hello ($$$$$$)\n")
	assert.Equal(t, expected, trimTimestamp(log))
}

func TestReplaceAgentBuildVairables(t *testing.T) {
	setUp(t)
	defer tearDown()

	goServer.SendBuild(AgentId, buildId,
		protocol.EchoCommand("hello ${agent.location}"),
		protocol.EchoCommand("hello ${agent.hostname}"),
	)
	assert.Equal(t, "agent Building", stateLog.Next())
	assert.Equal(t, "build Passed", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())

	log, err := goServer.ConsoleLog(buildId)
	assert.Nil(t, err)
	config := GetConfig()
	expected := Sprintf("hello %v\nhello %v\n", config.WorkingDir, config.Hostname)
	assert.Equal(t, expected, trimTimestamp(log))
}

func TestReplaceDateBuildVairables(t *testing.T) {
	setUp(t)
	defer tearDown()

	goServer.SendBuild(AgentId, buildId, protocol.EchoCommand("${date}"))
	assert.Equal(t, "agent Building", stateLog.Next())
	assert.Equal(t, "build Passed", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())

	log, err := goServer.ConsoleLog(buildId)
	assert.Nil(t, err)
	log = strings.TrimSpace(trimTimestamp(log))
	_, err = time.Parse("2006-01-02 15:04:05 PDT", log)
	assert.Nil(t, err)
}

func TestShowWarningInfoWhenThereIsUnsupportedBuildCommand(t *testing.T) {
	setUp(t)
	defer tearDown()

	cmd := protocol.NewBuildCommand("fancy")
	goServer.SendBuild(AgentId, buildId, cmd)
	assert.Equal(t, "agent Building", stateLog.Next())
	assert.Equal(t, "build Passed", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())

	log, err := goServer.ConsoleLog(buildId)
	assert.Nil(t, err)

	expected := Sprintf("WARN: Golang Agent does not support build comamnd 'fancy'")
	assert.True(t, strings.HasPrefix(trimTimestamp(log), expected), "console log must start with: %v", expected)
}

func TestShouldFailBuildIfWorkingDirIsSetToOutsideOfAgentWorkingDir(t *testing.T) {
	setUp(t)
	defer tearDown()

	goServer.SendBuild(AgentId, buildId,
		echo("echo hello world").Setwd("../../../"),
	)
	assert.Equal(t, "agent Building", stateLog.Next())
	assert.Equal(t, "build Failed", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())
}
