// Copyright (c) The Farseek Authors
// SPDX-License-Identifier: MPL-2.0

package farseek

type CallbackUIOutput struct {
	OutputFn func(string)
}

func (o *CallbackUIOutput) Output(v string) {
	o.OutputFn(v)
}
