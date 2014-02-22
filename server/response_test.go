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

package pubsubsql

import "testing"
import "encoding/json"

//import "fmt"

func validateResponseJSON(t *testing.T, res response) {
	var v interface{}
	netbytes, _ := res.toNetworkReadyJSON()
	bytes := fromNetworkBytes(netbytes)
	err := json.Unmarshal(bytes, &v)
	if err != nil {
		t.Error("failed to validate JSONBuilder:", err)
		t.Error(string(bytes))
	}
}

func TestErrorResponseJSON(t *testing.T) {
	res := &errorResponse{msg: "test error message"}
	validateResponseJSON(t, res)
}

func TestOkResponseJSON(t *testing.T) {
	res := &okResponse{}
	validateResponseJSON(t, res)
}

