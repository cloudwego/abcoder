/**
 * Copyright 2025 ByteDance Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package utils

import (
	"bytes"
	"context"
	"os/exec"

	"github.com/cloudwego/abcoder/lang/log"
)

func ExecCmdWithInstall(ctx context.Context, cmd string, args []string, installCmd string, installArgs []string) error {
	_, err := exec.LookPath(cmd)
	if err != nil {
		if installCmd == "" {
			return err
		}
		log.Info("install %s", installCmd)
		cmd := exec.CommandContext(ctx, installCmd, installArgs...)
		buf := bytes.NewBuffer(nil)
		cmd.Stdout = buf
		cmd.Stderr = buf
		if err = cmd.Run(); err != nil {
			log.Info("install %s failed, %s", installCmd, buf.String())
			return err
		}
	}
	exe := exec.CommandContext(ctx, cmd, args...)
	buf := bytes.NewBuffer(nil)
	exe.Stdout = buf
	exe.Stderr = buf
	if err = exe.Run(); err != nil {
		log.Info("exec %s failed, %s", cmd, buf.String())
		return err
	}
	return nil
}
