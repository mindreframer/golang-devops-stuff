// Package tasking provides support for running tasks for gotask.
// It is intended to be used in concert with the "gotask" command,
// which runs any function of the form
//     func TaskXxx(*tasking.T)
// where Xxx can be any alphanumeric string (but the first letter must not
// be in [a-z]) and serves to identify the task name.
// These TestXxx functions must be declared inside a GOPATH so that gotask can 
// find and compile it.
//
// Task Definition
//
// A task is defined in the form of
//     // +build gotask
//  
//     package main
//  
//     import "github.com/jingweno/gotask/tasking"
//  
//     // NAME
//     //    The name of the task - a one-line description of what it does
//     //
//     // DESCRIPTION
//     //    A textual description of the task function
//     //
//     // OPTIONS
//     //    Definition of what command line options it takes
//     func TaskXxx(t *tasking.T) {
//         ...
//     }
//
// The comments for the task function are parsed as the
// task's man page by following the man page layout: Section NAME
// contains the name of the task and a one-line description of what it
// does, separated by a "-"; Section DESCRIPTION contains the textual
// description of the task function; Section OPTIONS contains the
// definition of the command line flags it takes.
// By default, gotask dasherizes the Xxx part of the task function
// name and use it as the task name if there's no task name declared in
// the comment.
// The gotask build tag constraints task functions to gotask build only.
// Without the build tag, task functions will be available to application build
// which may not be desired.
//
// Flags
// 
// Flags are declared in section OPTIONS of the task function man page in the comments.
// The definition of flags should follow the POSIX convention, see
// http://www.gnu.org/software/libc/manual/html_node/Argument-Syntax.html.
//
// For bool flag, the format is
//     -SHORT-NAME, --LONG-NAME
//         DESCRIPTION 
// For string flag, the format is
//     -SHORT-NAME, --LONG-NAME=<VALUE>
//         DESCRIPTION
// If the string flag has a default value, remove the enclosing "<" and ">":
//     -SHORT-NAME, --LONG-NAME=VALUE
//         DESCRIPTION
//
// See https://github.com/jingweno/gotask/tree/master/examples for examples.
package tasking
