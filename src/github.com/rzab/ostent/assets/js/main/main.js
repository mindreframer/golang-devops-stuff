var stateClass = React.createClass({
    getInitialState: function() { return {V: this.props.initialValue}; },
    render: function() {
	return (
	    React.DOM.span(null, this.state.V, this.props.append)
	);
    }
});
var percentClass = React.createClass({
    getInitialState: function() { return {V: this.props.initialValue}; },
    render: function() {
	return (
	    React.DOM.span({className:label_colorPercent(this.state.V)}, this.state.V, '%')
	);
    }
});

function newState(ID) {
    var node = document.getElementById(ID);
    n = new stateClass({
	elementID:    node,
	initialValue: node.innerHTML
    });
    React.renderComponent(n, n.props.elementID);
    return n;
}
function newPercent(ID) {
    var node = document.getElementById(ID);
    n = new percentClass({
	elementID:    node,
	initialValue: '',
    });
    // React.renderComponent(n, n.props.elementID);
    return n;
}

function update(Data) { // assert window["WebSocket"]
    $('span .tooltipable').tooltip();

    // https://stackoverflow.com/q/15725717
    $('[data-parent="#disk-accordion"]').on('click',function(e) {
	if ($($(this).attr('href')).hasClass('in')){
            e.stopPropagation();
	}
    });

    var proc_table   = ProcTable(null);        // from gen/build.js
    var disk_table   = DiskTable(null);        // from gen/build.js
    var inodes_table = DiskInodesTable(null);  // from gen/build.js
    var cpu_table    = CPUTable(null);         // from gen/build.js
    var ifs_table    = Interfaces(null);       // from gen/build.js
    // console.log(React.renderComponentToString(Interfaces(null)));
    React.renderComponent(proc_table,   document.getElementById('ps-table'));
    React.renderComponent(disk_table,   document.getElementById('df-table'));
    React.renderComponent(inodes_table, document.getElementById('dfi-table'));
    React.renderComponent(cpu_table,    document.getElementById('cpu-table'));
    React.renderComponent(ifs_table,    document.getElementById('ifs-table'));

    var onmessage = onmessage = function(event) {
	var data = JSON.parse(event.data);

	proc_table  .setState(data.ProcTable);
	disk_table  .setState(data.DiskTable);
	inodes_table.setState(data.DiskTable);

	cpu_table   .setState(data.CPU);
	ifs_table   .setState({List: data.Interfaces});

	Data.About.Hostname  .setState({V: data.About.Hostname  });
	Data.About.IP        .setState({V: data.About.IP        });
	Data.System.Uptime   .setState({V: data.System.Uptime   });
	Data.System.LA       .setState({V: data.System.LA       });

	Data.RAM.Free        .setState({V: data.RAM.Free        });
	Data.RAM.Used        .setState({V: data.RAM.Used        });
	Data.RAM.Total       .setState({V: data.RAM.Total       });

	React.renderComponent(Data.RAM.UsePercent, Data.RAM.UsePercent.props.elementID);
	Data.RAM.UsePercent  .setState({V: data.RAM.UsePercent  });

	Data.Swap.Free       .setState({V: data.Swap.Free       });
	Data.Swap.Used       .setState({V: data.Swap.Used       });
	Data.Swap.Total      .setState({V: data.Swap.Total      });

    	React.renderComponent(Data.Swap.UsePercent, Data.Swap.UsePercent.props.elementID);
	Data.Swap.UsePercent .setState({V: data.Swap.UsePercent });
    };

    var news = function() {
	var conn = new WebSocket("ws://" + HTTP_HOST + "/ws");
	var again = function() {
	    $("a.state").unbind('click');
	    window.setTimeout(news, 5000);
	};
	conn.onclose = again;
	conn.onerror = again;
	conn.onmessage = onmessage;

	conn.onopen = function() {
	    conn.send(location.search);
	    $(window).bind('popstate', function() {
		conn.send(location.search);
	    });
	};

	$("a.state").click(function() {
	    history.pushState({path: this.path}, '', this.href)
	    conn.send(this.search);
	    return false;
	});
    };
    news();
}

function ready() {
    update({
	About: { Hostname:   newState('Data.About.Hostname')
		 , IP:       newState('Data.About.IP')
	       },
	System: { Uptime:    newState('Data.System.Uptime')
		  , LA:      newState('Data.System.LA')
		},
	RAM: { Free:         newState('Data.RAM.Free')
	       , Used:       newState('Data.RAM.Used')
	       , UsePercent: newPercent('Data.RAM.UsePercent')
	       , Total:      newState('Data.RAM.Total')
	     },
	Swap: { Free:         newState('Data.Swap.Free')
		, Used:       newState('Data.Swap.Used')
		, UsePercent: newPercent('Data.Swap.UsePercent')
		, Total:      newState('Data.Swap.Total')
	      }
    });
}
