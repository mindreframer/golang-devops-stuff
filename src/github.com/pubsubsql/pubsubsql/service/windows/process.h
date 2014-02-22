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

#ifndef PUBSUBSQLSVC_PROCESS_H
#define PUBSUBSQLSVC_PROCESS_H

#include <memory>
#include <thread>
#include "pipe.h"

class process {
public:
	process(); 
	~process();
	bool start(char* commandLine, const char* eventSource);
	void stop();
	void wait(unsigned milliseconds);
	HANDLE handle();

private:
	PROCESS_INFORMATION processInfo; 
	pipe stderrPipe;
	pipe stdinPipe;
	std::thread logThread;

};

#endif //PUBSUBSQLSVC_PROCESS_H