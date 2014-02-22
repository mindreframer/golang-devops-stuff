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


#include "eventlog.h"
#include "pubsubsqllog.h"
#include <iostream>

std::string eventlog::getPath() {
	const DWORD MAX_FILE_PATH = 4096;
	char fileName[1 + MAX_FILE_PATH];
	DWORD retCode = GetModuleFileName(static_cast<HMODULE>(0),	fileName, MAX_FILE_PATH);
	if (0 == retCode) {
		std::cerr << "Failed to retrieve module name" << std::endl;
		return std::string("");
	}
	if (MAX_FILE_PATH == retCode) {
		std::cerr << "Module path is too large" << std::endl;
		return std::string("");
	}
	fileName[MAX_FILE_PATH] = '\0'; // guard
	std::string path(fileName);
	size_t eraseFrom = 1 + path.find_last_of('\\');
	if (path.size() >=  eraseFrom) {
		path.erase(eraseFrom);
	}
	return path;
}

bool eventlog::install(const char* messagedll, const char* eventSource) {
	const char* SYSLOG_PATH = "SYSTEM\\CurrentControlSet\\Services\\Eventlog\\Application\\";
	const char* VALUE_NAME = "EventMessageFile";
	//
	std::string value = getPath();
	value.append(messagedll);
	//value.append(".dll");
	//
	std::string key(SYSLOG_PATH);
	key.append(eventSource);
	// update registry
	HKEY handleKey;
	DWORD retCode = RegCreateKeyEx(
		HKEY_LOCAL_MACHINE,			// handle to an open registry key
		key.c_str(),				// name of a subkey
		0,							// reserved and must be zero
		0,							// user-defined class type of this key
		REG_OPTION_NON_VOLATILE,	// options
		KEY_READ | KEY_WRITE |
		KEY_QUERY_VALUE,			// access rights for the key
		0,							// default security descriptor
		&handleKey,					// receives a handle to the key
		0							// receives: new key or existed
		);
	if (ERROR_SUCCESS != retCode) {
		std::cerr << "Failed to open registry" << std::endl;
		return false;
	}
	retCode = RegSetValueEx(
		handleKey,				// handle to an open registry key
		VALUE_NAME,				// name of the value to be set
		0,						// reserved and must be zero
		REG_SZ,					// type of data pointed by the data parameter
		(LPBYTE)value.c_str(),	// data to be stored
		static_cast<DWORD>(value.size() + 1)// data parameter size (with zero)
		);
	RegCloseKey(handleKey);
	if (ERROR_SUCCESS != retCode) {
		std::cerr << "Failed to set value in the registry" << std::endl;
		return false;
	}
	return true;
}

eventlog::eventlog(const char* eventSource) {
	eventSourceHandle = RegisterEventSource(NULL, eventSource);
	if (NULL == eventSourceHandle) {
		std::cerr << "Failed to open event log" << std::endl;	
	}
}

eventlog::~eventlog() {
	CloseHandle(eventSourceHandle);
}

void eventlog::logdebug(const char* message) {
	log(message, EVENTLOG_INFORMATION_TYPE);
}

void eventlog::loginfo(const char* message) {
	log(message, EVENTLOG_INFORMATION_TYPE);
}

void eventlog::logwarn(const char* message) {
	log(message, EVENTLOG_WARNING_TYPE);
}

void eventlog::logerror(const char* message) {
	log(message, EVENTLOG_ERROR_TYPE);
}

void eventlog::log(const char* message, WORD messageType) {
	ReportEvent(eventSourceHandle, messageType, 0, MSG_SYSLOG, 0, 1, 0, &message, 0);
}
