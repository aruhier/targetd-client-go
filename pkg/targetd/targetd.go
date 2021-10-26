/*
 * Copyright 2021 Johan Fleury <jfleury@arcaik.net>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package targetd

import (
	"fmt"

	"github.com/powerman/rpc-codec/jsonrpc2"
)

const (
	InvalidError              = -1
	NameConflictError         = -50
	CloneNameExistsError      = -51
	InitiatorExistsError      = -52
	FSExistsError             = -53
	VolumeNotFoundError       = -103
	FSNotFoundError           = -104
	InvalidPoolError          = -110
	SSNotFoundError           = -112
	VolumeExportNotFoundError = -151
	VolumeGroupNotFoundError  = -152
	NotSupportedError         = -153
	AccessGroupNotFoundError  = -200
	UnexpectedExitCodeError   = -303
	VolumeMaskedError         = -303
	NFSExportNotFoundError    = -400
	NFSNotSupportedError      = -401
	NoFreeHostLunIdError      = -1000
	InvalidArgumentError      = -32602
)

// Error is a wrapper struct arround jsonrpc2.Error that helps getting
// information about server errors.
type Error struct {
	inner *jsonrpc2.Error
}

// UnwrapError transform an error into an Error if possible.
func UnwrapError(err error) *Error {
	inner := jsonrpc2.ServerError(err)

	if inner == nil {
		return nil
	}

	return &Error{inner: inner}
}

// String returns a formated error.
func (e *Error) String() string {
	return fmt.Sprintf("%s (code: %d)", e.Message(), e.Code())
}

// Message exposes the error message returned by the server.
func (e *Error) Message() string {
	return e.inner.Message
}

// Code exposes the error code returned by the server.
func (e *Error) Code() int {
	return e.inner.Code
}

type Pool struct {
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	FreeSize int64  `json:"free_size"`
	Type     string `json:"type"`
	UUID     string `json:"uuid"`
}

type Volume struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	UUID string `json:"uuid"`
}

type Export struct {
	InitiatorWwn string `json:"initiator_wwn"`
	LUN          int32  `json:"lun"`
	VolName      string `json:"vol_name"`
	VolSize      int64  `json:"vol_size"`
	VolUUID      string `json:"vol_uuid"`
	Pool         string `json:"pool"`
}

type Client struct {
	jsonrpc *jsonrpc2.Client
}

func New(url string) *Client {
	return &Client{
		jsonrpc: jsonrpc2.NewHTTPClient(url),
	}
}

func (c *Client) PoolList() ([]Pool, error) {
	var pools []Pool

	if err := c.jsonrpc.Call("pool_list", nil, &pools); err != nil {
		return nil, err
	}

	return pools, nil
}

func (c *Client) VolList(pool string) ([]Volume, error) {
	var volumes []Volume
	args := map[string]string{
		"pool": pool,
	}

	if err := c.jsonrpc.Call("vol_list", args, &volumes); err != nil {
		return nil, err
	}

	return volumes, nil
}

func (c *Client) VolCreate(pool, name string, size int64) error {
	args := map[string]interface{}{
		"pool": pool,
		"name": name,
		"size": size,
	}

	return c.jsonrpc.Call("vol_create", args, nil)
}

func (c *Client) VolCopy(pool, orig, name string, size int64) error {
	args := map[string]interface{}{
		"pool":     pool,
		"vol_orig": orig,
		"vol_new":  name,
		"size":     size,
	}

	return c.jsonrpc.Call("vol_resize", args, nil)
}

func (c *Client) VolResize(pool, name string, size int64) error {
	args := map[string]interface{}{
		"pool": pool,
		"name": name,
		"size": size,
	}

	return c.jsonrpc.Call("vol_resize", args, nil)
}

func (c *Client) VolDestroy(pool, name string) error {
	args := map[string]string{
		"pool": pool,
		"name": name,
	}

	return c.jsonrpc.Call("vol_destroy", args, nil)
}

func (c *Client) ExportList() ([]Export, error) {
	var exports []Export

	if err := c.jsonrpc.Call("export_list", nil, &exports); err != nil {
		return nil, err
	}

	return exports, nil
}

func (c *Client) ExportCreate(pool, vol, initiatorWWN string, lun int32) error {
	args := map[string]interface{}{
		"pool":          pool,
		"vol":           vol,
		"initiator_wwn": initiatorWWN,
		"lun":           lun,
	}

	return c.jsonrpc.Call("export_create", args, nil)
}

func (c *Client) ExportDestroy(pool, vol, initiatorWWN string) error {
	args := map[string]string{
		"pool":          pool,
		"vol":           vol,
		"initiator_wwn": initiatorWWN,
	}

	return c.jsonrpc.Call("export_destroy", args, nil)
}

func (c *Client) GetFirstAvailableLun() (int32, error) {
	luns := map[int32]bool{}

	exports, err := c.ExportList()
	if err != nil {
		return -1, err
	}

	for _, export := range exports {
		luns[export.LUN] = true
	}

	for i := int32(0); i < 255; i++ {
		if _, ok := luns[i]; !ok {
			return i, nil
		}
	}

	return -1, fmt.Errorf("No LUN available, target already have 255 LUNs allocated")
}
