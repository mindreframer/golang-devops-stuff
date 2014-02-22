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

#ifndef PUBSUBSQLSVC_EVENTLOG_H
#define PUBSUBSQLSVC_EVENTLOG_H

#include <Windows.h>
#include <string>

class eventlog {
public:
	static std::string getPath();
	static bool install(const char* syslogdll, const char* syslogname);
	eventlog(const char* syslogname);	
	~eventlog();

	void logdebug(const char* message);
	void loginfo(const char* message);
	void logwarn(const char* message);
	void logerror(const char* message);

private:
	void log(const char* message, WORD messageType);

	HANDLE eventSourceHandle;
};

#endif //PUBSUBSQLSVC_EVENTLOG_H
