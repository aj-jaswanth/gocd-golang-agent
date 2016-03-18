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
	"fmt"
	. "github.com/gocd-contrib/gocd-golang-agent/agent"
	"github.com/gocd-contrib/gocd-golang-agent/protocal"
	"github.com/xli/assert"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestUploadArtifactFile(t *testing.T) {
	setUp(t)
	defer tearDown()

	artifactWd := createPipelineDir()

	fname := "artifact.txt"
	err := writeFile(artifactWd, fname)
	assert.Nil(t, err)

	goServer.SendBuild(AgentId, buildId,
		protocal.UploadArtifactCommand(fname, "").Setwd(artifactWd))

	assert.Equal(t, "agent Building", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())

	log, err := goServer.ConsoleLog(buildId)
	assert.Nil(t, err)
	expected := fmt.Sprintf("Uploading artifacts from %v/%v to [defaultRoot]\n", artifactWd, fname)
	assert.Equal(t, expected, trimTimestamp(log))

	f := goServer.ArtifactFile(buildId, fname)
	finfo, err := os.Stat(f)
	assert.Nil(t, err)
	assert.Equal(t, fname, finfo.Name())

	content, err := ioutil.ReadFile(f)
	assert.Nil(t, err)
	assert.Equal(t, "file created for test", string(content))

	checksum, err := goServer.Checksum(buildId)
	assert.Nil(t, err)
	assert.True(t, contains(checksum, fname+"="), "checksum: %v", checksum)
}

func writeFile(dir, fname string) error {
	err := os.MkdirAll(dir, 0744)
	if err != nil {
		return err
	}
	fpath := filepath.Join(dir, fname)
	f, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE, 0744)
	if err != nil {
		return err
	}
	data := []byte("file created for test")
	n, err := f.Write(data)
	if err == nil && n < len(data) {
		return io.ErrShortWrite
	}
	return f.Close()
}