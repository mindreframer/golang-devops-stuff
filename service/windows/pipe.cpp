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
#include <Windows.h>
#include <io.h>
#include "pipe.h"

pipe::pipe()
:	valid(false)
,	readFile(nullptr)		
,	writeFile(nullptr)		
,	pipeHandle(INVALID_HANDLE_VALUE)
,	readHandle(INVALID_HANDLE_VALUE)
,	writeHandle(INVALID_HANDLE_VALUE)
{
	// init security attributes
	SECURITY_ATTRIBUTES securityAttributes;
	initSecurityAttributes(securityAttributes);
	// create os pipe
	if (!CreatePipe(&readHandle, &writeHandle, &securityAttributes, BUFFER_SIZE)) {
		std::cerr << "pipe error: CreatePipe failed err:" << GetLastError() << std::endl;	
		return;
	}
	// convert pipe read/write handle to cruntime FILE *
	readFile = toFileFromHandle(readHandle, "r");	
	writeFile = toFileFromHandle(writeHandle, "w");	
	if (nullptr == readFile || nullptr == writeFile) {
		std::cerr << "pipe error: Failed to convert file handles " << GetLastError() << std::endl;	
		return;
	}
	// 
	valid = true;
}

pipe::~pipe() {
	CloseHandle(pipeHandle);
	CloseHandle(readHandle);
	CloseHandle(writeHandle);
}

bool pipe::ok() {
	return valid;
}

const char* pipe::readLine() {
	return fgets(buffer, BUFFER_SIZE, readFile);
}

void pipe::writeLine(const char* line) {
	WriteFile(writeHandle, line, (DWORD)strlen(line), 0, NULL);
	WriteFile(writeHandle, "\n", 1, 0, NULL);
}
	
FILE* pipe::toFileFromHandle(HANDLE handle, const char* fileOpenMode) {
	int cfileDescriptor = _open_osfhandle((intptr_t)handle, 0);
	if (cfileDescriptor == -1) return nullptr;
	return _fdopen(cfileDescriptor, fileOpenMode);
}

HANDLE pipe::getWriteHandle() {
	return writeHandle;
}

HANDLE pipe::getReadHandle() {
	return readHandle;
}

void pipe::initSecurityAttributes(SECURITY_ATTRIBUTES& securityAttributes) {
	securityAttributes.nLength = sizeof(securityAttributes);
	securityAttributes.lpSecurityDescriptor = NULL;
	securityAttributes.bInheritHandle = TRUE;
}
