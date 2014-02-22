/* Copyright (C) 2013 CompleteDB LLC.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with PubSubSQL.  If not, see <http://www.gnu.org/licenses/>.
 */

#include <iostream>
#include "process.h"
#include "eventlog.h"

process::process() {
	ZeroMemory(&processInfo, sizeof(processInfo));
}

process::~process() {
	CloseHandle(processInfo.hProcess);
	CloseHandle(processInfo.hThread);
}

void redirectLogEntries(pipe& stderrPipe) {

}

bool process::start(char* commandLine, const char* eventSource) {
	//
	SECURITY_ATTRIBUTES securityAttributes;
	pipe::initSecurityAttributes(securityAttributes);
	// setup startup info
	STARTUPINFO startupInfo;
	ZeroMemory(&startupInfo, sizeof(startupInfo));
	startupInfo.cb = sizeof(startupInfo);
	startupInfo.dwFlags |= STARTF_USESTDHANDLES;
	startupInfo.hStdInput = stdinPipe.getReadHandle();
	startupInfo.hStdError = stderrPipe.getWriteHandle();
	startupInfo.hStdOutput = NULL;
	// create pubsubsql.exe process	
	BOOL created = CreateProcess(NULL, commandLine, NULL, NULL, TRUE, CREATE_NO_WINDOW, NULL, NULL, &startupInfo, &processInfo); 
	if (!created) {
		std::cerr << "process error: Failed to create child process" << std::endl;
		return false;
	}
	// redirect log entries from pubsubsql to event log
	std::thread t ( [] (pipe& stderrPipe, const char* eventSource) {
		eventlog log(eventSource);
		for (;;) {
			const char* line = stderrPipe.readLine();
			if (!line) {
				std::cerr << "failed to read line" << std::endl;	
				return;
			}
			std::cout << line;
			// redirect log message to event log
			if (strncmp(line, "info", 4) == 0) {
				log.loginfo(line);
			} else if (strncmp(line, "error", 5) == 0) {
				log.logerror(line);
			} else if (strncmp(line, "debug", 5) == 0) {
				log.loginfo(line);
			} else {
				log.logwarn(line);
			}
		}
	}, std::ref(stderrPipe), eventSource);
	logThread = std::move(t);
	// 
	return true;
}

void process::stop() {
	stdinPipe.writeLine("q");
}

void process::wait(unsigned milliseconds) {
	WaitForSingleObject(processInfo.hProcess, milliseconds);
	// let the last log entry to go through
	Sleep(100);
	// just in case
	TerminateProcess(processInfo.hProcess, EXIT_SUCCESS);
	TerminateThread(logThread.native_handle(), EXIT_SUCCESS);
	logThread.detach();	
}
	
HANDLE process::handle() {
	return processInfo.hProcess;
}
