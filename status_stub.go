//go:build !linux

package rcebot

import "context"

type statusNotifier struct{}

func newStatusNotifier(context.Context) statusNotifier { return statusNotifier{} }

func (statusNotifier) IsValid() bool         { return false }
func (statusNotifier) Close() error          { return nil }
func (statusNotifier) Ready()                {}
func (statusNotifier) Stopping()             {}
func (statusNotifier) Reloading()            {}
func (statusNotifier) ExtendTimeout() func() { return func() {} }
