/** @jsx React.DOM */

/* Interfaces */
var Interfaces = React.createClass({displayName: 'Interfaces',
    getInitialState: function() { return Data_Interfaces; },

    render: function() {
	var ifs_rows  = this.state.List.map(function(interface){
	    return (

React.DOM.tr( {key:"if" + interface.Name}, 
React.DOM.td(null, interface.Name),
React.DOM.td( {className:"digital"}, interface.DeltaIn),
React.DOM.td( {className:"digital"}, interface.DeltaOut),
React.DOM.td( {className:"digital"}, interface.In),
React.DOM.td( {className:"digital"}, interface.Out)
)

	    );
	});
	return (
React.DOM.table( {className:"table1 stripe-table"}, 
	React.DOM.thead(null, 
		React.DOM.tr(null, 
			React.DOM.th(null, "Interface"),
			React.DOM.th( {className:"digital nobr"}, 
				"In",
				React.DOM.span( {className:"unit"}, "ps")
			),
			React.DOM.th( {className:"digital nobr"}, 
				"Out",
				React.DOM.span( {className:"unit"}, "ps")
			),
			React.DOM.th( {className:"digital nobr"}, 
				"In",
				React.DOM.span( {className:"unit"}, "%4G")
			),
			React.DOM.th( {className:"digital nobr"}, 
				"Out",
				React.DOM.span( {className:"unit"}, "%4G")
			)
		)
	),
	React.DOM.tbody(null, 
		ifs_rows
	)
)

	);
    }
});

/* CPUTable */
function label_colorPercent(p) {
    return "label label-"+ _colorPercent(p);
}
function text_colorPercent(p) {
    return "text-"+ _colorPercent(p);
}
function _colorPercent(p) {
	if (p > 90) {
		return "danger";
	}
	if (p > 80) {
		return "warning";
	}
	if (p > 20) {
		return "info";
	}
	return "success";
}
function diskname(disk) {
  if (disk.ShortDiskName === "") {
    return (React.DOM.span(null, disk.DiskName));
  }
  var span = (
  React.DOM.span( {className:"tooltipable",
     'data-toggle':"tooltip",
     'data-placement':"left",
     title:disk.DiskName}, disk.ShortDiskName,"...")
);
  $('span .tooltipable').tooltip();
  return span;
}

var CPUTable = React.createClass({displayName: 'CPUTable',
    getInitialState: function() { return Data_CPU; },

    render: function() {
	var cpu_row0 = function(totalCPU) {
	    return (



React.DOM.tr( {key:"cpu-1"}, 
React.DOM.td( {className:"digital nobr"}, 
	React.DOM.span( {id:"Data.CPU.N"}, totalCPU.N, " total")
),
React.DOM.td( {className:"digital"}, React.DOM.span( {id:"Data.CPU.User", className:text_colorPercent(totalCPU.User)}, totalCPU.User)),
React.DOM.td( {className:"digital"}, React.DOM.span( {id:"Data.CPU.Sys", className:text_colorPercent(totalCPU.Sys)}, totalCPU.Sys)),
React.DOM.td( {className:"digital"}, React.DOM.span( {id:"Data.CPU.Idle", className:text_colorPercent(100 - totalCPU.Idle)}, totalCPU.Idle))
)


	    );
	}(this.state);
	var cpu_rows  = this.state.List.map(function(cpu){
	    return (



React.DOM.tr( {key:"cpu" + cpu.N}, 
React.DOM.td( {className:"digital"}, "#",cpu.N),
React.DOM.td( {className:"digital"}, React.DOM.span( {id:"core0.User", className:text_colorPercent(cpu.User)}, cpu.User)),
React.DOM.td( {className:"digital"}, React.DOM.span( {id:"core0.Sys", className:text_colorPercent(cpu.Sys)}, cpu.Sys)),
React.DOM.td( {className:"digital"}, React.DOM.span( {id:"core0.Idle", className:text_colorPercent(100 - cpu.Idle)}, cpu.Idle))
)


	    );
	});
	return (
React.DOM.table( {className:"table1 stripe-table"}, 
	React.DOM.thead(null, 
		React.DOM.tr(null, 
			React.DOM.th(null),
			React.DOM.th( {className:"digital"}, "User%"),
			React.DOM.th( {className:"digital"}, "Sys%"),
			React.DOM.th( {className:"digital"}, "Idle%")
		)
	),
	React.DOM.tbody(null, 
		cpu_row0,cpu_rows
	)
)

	);
    }
});


/* DiskTable Inodes */
var DiskInodesTable = React.createClass({displayName: 'DiskInodesTable',
//  getInitialState: function() { return {List: Data_DiskTable.List, table_links: Data_DiskTable.Links}; },
    getInitialState: function() { return Data_DiskTable; },

    render: function() {
	var the_rows ;
    if (this.state.List != null) {
	  the_rows = this.state.List.map(function(disk){
	    return (


React.DOM.tr( {key:"inode" + disk.DirName}, 
React.DOM.td(null, diskname(disk)),
React.DOM.td( {className:"digital"}, disk.Ifree),
React.DOM.td( {className:"digital"}, 
	disk.Iused," ",
	React.DOM.sup(null, React.DOM.span( {className:label_colorPercent(disk.IusePercent)}, disk.IusePercent,"%"))
),
React.DOM.td( {className:"digital"}, disk.Inodes)
)

	    );
	  });
    }
	return (
React.DOM.table( {className:"table1 stripe-table"}, 
	React.DOM.thead(null, 
		React.DOM.tr(null, 
			React.DOM.th( {className:"header"},         "        Device"),
			React.DOM.th( {className:"header digital"}, "Avail"),
			React.DOM.th( {className:"header digital"}, "Used"),
			React.DOM.th( {className:"header digital"}, "Total")
		)
	),
	React.DOM.tbody(null, 
		the_rows
	)
)

	);
    }
});

/* DiskTable Space */
var DiskTable = React.createClass({displayName: 'DiskTable',
//  getInitialState: function() { return {List: Data_DiskTable.List, table_links: Data_DiskTable.Links}; },
    getInitialState: function() { return Data_DiskTable; },

    render: function() {
	var links = this.state.Links;
    var the_rows ;
	if (this.state.List != null) {
	  the_rows = this.state.List.map(function(disk){
	    return (


React.DOM.tr( {key:"space" + disk.DirName}, 
React.DOM.td(null, diskname(disk)),
React.DOM.td(null, disk.DirName),
React.DOM.td( {className:"digital"}, disk.Avail),
React.DOM.td( {className:"digital"}, 
	disk.Used," ",
	React.DOM.sup(null, React.DOM.span( {className:label_colorPercent(disk.UsePercent)}, disk.UsePercent,"%"))
),
React.DOM.td( {className:"digital"}, disk.Total)
)

	    );
	  });
    }
	return (
React.DOM.table( {className:"table1 stripe-table"}, 
	React.DOM.thead(null, 
		React.DOM.tr(null, 
			React.DOM.th( {className:"header"},         "        ",        React.DOM.a( {href:links.DiskName.Href, className:links.DiskName.Class}, "Device")),
			React.DOM.th( {className:"header"},         "        ",        React.DOM.a( {href:links.DirName.Href,  className:links.DirName.Class}, "Mounted")),
			React.DOM.th( {className:"header digital"}, React.DOM.a( {href:links.Avail.Href,    className:links.Avail.Class}, "Avail")),
			React.DOM.th( {className:"header digital"}, React.DOM.a( {href:links.Used.Href,     className:links.Used.Class}, "Used")),
			React.DOM.th( {className:"header digital"}, React.DOM.a( {href:links.Total.Href,    className:links.Total.Class}, "Total"))
		)
	),
	React.DOM.tbody(null, 
		the_rows
	)
)

	);
    }
});

/* ProcTable */
var ProcTable = React.createClass({displayName: 'ProcTable',
    getInitialState: function() { return Data_ProcTable; },

    render: function() {
	var links = this.state.Links;
	var ps_rows  = this.state.List.map(function(proc){
	    return (

React.DOM.tr( {key:"pid" + proc.PID}, 
React.DOM.td( {className:"digital"}, proc.PID),
React.DOM.td( {className:"digital"}, proc.User),
React.DOM.td( {className:"digital"}, proc.Priority),
React.DOM.td( {className:"digital"}, proc.Nice),
React.DOM.td( {className:"digital"}, proc.Size),
React.DOM.td( {className:"digital"}, proc.Resident),
React.DOM.td( {className:"center"}, proc.Time),
React.DOM.td(null, proc.Name)
)

	    );
	});
	return (
React.DOM.table( {className:"table1 stripe-table"}, 
	React.DOM.thead(null, 
		React.DOM.tr(null, 
			React.DOM.th( {className:"header digital"}, React.DOM.a( {href:links.PID.Href,      className:links.PID.Class}, "PID")),
			React.DOM.th( {className:"header digital"}, React.DOM.a( {href:links.User.Href,     className:links.User.Class}, "USER")),
			React.DOM.th( {className:"header digital"}, React.DOM.a( {href:links.Priority.Href, className:links.Priority.Class}, "PR")),
			React.DOM.th( {className:"header digital"}, React.DOM.a( {href:links.Nice.Href,     className:links.Nice.Class}, "NI")),
			React.DOM.th( {className:"header digital"}, React.DOM.a( {href:links.Size.Href,     className:links.Size.Class}, "VIRT")),
			React.DOM.th( {className:"header digital"}, React.DOM.a( {href:links.Resident.Href, className:links.Resident.Class}, "RES")),
			React.DOM.th( {className:"header center"},  " ", React.DOM.a( {href:links.Time.Href,     className:links.Time.Class}, "TIME")),
			React.DOM.th( {className:"header"},         "        ",        React.DOM.a( {href:links.Name.Href,     className:links.Name.Class}, "COMMAND"))
		)
	),
	React.DOM.tbody( {id:"procrows"}, 
		ps_rows
	)
)

	);
    }
});
