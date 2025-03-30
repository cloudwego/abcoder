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

package uniast

type Writer interface {
	// write a module onto Options.OutDir.
	WriteModule(repo *Repository, modPath string) error

	// SplitImportsAndCodes will split the imports and codes from the src.
	// the src has only codes, just return the src.
	SplitImportsAndCodes(src string) (codes string, imports []Import, err error)

	// IdToImport converts the identity to import.
	IdToImport(id Identity) (Import, error)

	// PatchImports patches the imports into file content
	PatchImports(file *File) ([]byte, error)
}
