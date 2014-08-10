
@newwebsocket = (onmessage) ->
        conn = null
        sendSearch = (search) -> sendJSON({Search: search})
        sendClient = (client) ->
                console.log(JSON.stringify(client), 'sendClient')
                return sendJSON({Client: client})
        sendJSON = (obj) ->
                # 0 conn.CONNECTING
                # 1 conn.OPEN
                # 2 conn.CLOSING
                # 3 conn.CLOSED
                if !conn? ||
                   conn.readyState == conn.CLOSING ||
                   conn.readyState == conn.CLOSED
                        init()
                if !conn? ||
                   conn.readyState != conn.OPEN
                        console.log('Not connected, cannot send', obj)
                        return
                return conn.send(JSON.stringify(obj))
        init = () ->
                hostport = window.location.hostname + (if location.port then ':' + location.port else '')
                conn = new WebSocket('ws://' + hostport + '/ws')
                conn.onopen = () ->
                        sendSearch(location.search)
                        $(window).bind('popstate', (() ->
                                sendSearch(location.search)
                                return))
                        return

                statesel = 'table thead tr .header a.state'
                again = (e) ->
                        $(statesel).unbind('click')
                        window.setTimeout(init, 5000) if !e.wasClean
                        return

                conn.onclose   = again
                conn.onerror   = again
                conn.onmessage = onmessage

                $(statesel).click(() ->
                        history.pushState({path: @path}, '', @href)
                        sendSearch(@search)
                        return false)
                return

        init()
        return {
                sendClient: sendClient
                sendSearch: sendSearch
                close: () -> conn.close()
        }

@IFbytesCLASS = React.createClass
        getInitialState: () -> Data.IFbytes # a global Data
        render: () ->
                Data = {IFbytes: @state}
                return ifbytes_table(Data, (ifbytes_rows(Data, $if) for $if in Data?.IFbytes?.List ? []))

@IFerrorsCLASS = React.createClass
        getInitialState: () -> Data.IFerrors # a global Data
        render: () ->
                Data = {IFerrors: @state}
                return iferrors_table(Data, (iferrors_rows(Data, $if) for $if in Data?.IFerrors?.List ? []))

@IFpacketsCLASS = React.createClass
        getInitialState: () -> Data.IFpackets # a global Data
        render: () ->
                Data = {IFpackets: @state}
                return ifpackets_table(Data, (ifpackets_rows(Data, $if) for $if in Data?.IFpackets?.List ? []))

@DFbytesCLASS = React.createClass
        getInitialState: () -> {DFlinks: Data.DFlinks, DFbytes: Data.DFbytes} # a global Data
        render: () ->
                Data = @state
                return dfbytes_table(Data, (dfbytes_rows(Data, $disk) for $disk in Data?.DFbytes?.List ? []))

@DFinodesCLASS = React.createClass
        getInitialState: () -> {DFlinks: Data.DFlinks, DFinodes: Data.DFinodes} # a global Data
        render: () ->
                Data = @state
                return dfinodes_table(Data, (dfinodes_rows(Data, $disk) for $disk in Data?.DFinodes?.List ? []))

@MEMtableCLASS = React.createClass
        getInitialState: () -> Data.MEM # a global Data
        render: () ->
                Data = {MEM: @state}
                return mem_table(Data, (mem_rows(Data, $mem) for $mem in Data?.MEM?.List ? []))

@CPUtableCLASS = React.createClass
        getInitialState: () -> Data.CPU # a global Data
        render: () ->
                Data = {CPU: @state}
                return cpu_table(Data, (cpu_rows(Data, $core) for $core in Data?.CPU?.List ? []))

@PStableCLASS = React.createClass
        getInitialState: () -> {PStable: Data.PStable, PSlinks: Data.PSlinks} # a global Data
        render: () ->
                Data = @state
                return ps_table(Data, (ps_rows(Data, $proc) for $proc in Data?.PStable?.List ? []))

@VGtableCLASS = React.createClass
        getInitialState: () -> { # a global Data:
                VagrantMachines: Data.VagrantMachines
                VagrantError:    Data.VagrantError
                VagrantErrord:   Data.VagrantErrord
        }
        render: () ->
                Data = @state
                if Data?.VagrantErrord? and Data.VagrantErrord
                        rows = [vagrant_error(Data)]
                else
                        rows = (vagrant_rows(Data, $machine) for $machine in Data?.VagrantMachines?.List ? [])
                return vagrant_table(Data, rows)

@addNoscript = ($) -> $.append('<noscript />').find('noscript').get(0)

@HideClass = React.createClass
        statics: component: (opt) -> React.renderComponent(HideClass(opt), addNoscript(opt.$button_el))

        reduce: (data) ->
                if data?.Client?
                        value = data.Client[@props.key]
                        return {Hide: value} if value isnt undefined
                return null
        getInitialState: () -> @reduce(Data) # a global Data
        componentDidMount: () -> @props.$button_el.click(@click)
        render: () ->
                @props.$collapse_el.collapse(if @state.Hide then 'hide' else 'show')
                buttonactive =  @state.Hide
                buttonactive = !@state.Hide if @props.reverseActive? and @props.reverseActive
                @props.$button_el[if buttonactive then 'addClass' else 'removeClass']('active')
                return null
        click: (e) ->
                (S = {})[@props.key] = !@state.Hide
                websocket.sendClient(S)
                e.stopPropagation() # preserves checkbox/radio
                e.preventDefault()  # checked/selected state
                return undefined

@ButtonClass = React.createClass
        statics: component: (opt) -> React.renderComponent(ButtonClass(opt), addNoscript(opt.$button_el))

        reduce: (data) ->
                if data?.Client?
                        S = {}
                        S.Hide = data.Client[@props.Khide] if                   data.Client[@props.Khide] isnt undefined # Khide is a required prop
                        S.Able = data.Client[@props.Kable] if @props.Kable? and data.Client[@props.Kable] isnt undefined
                        S.Send = data.Client[@props.Ksend] if @props.Ksend? and data.Client[@props.Ksend] isnt undefined
                        S.Text = data.Client[@props.Ktext] if @props.Ktext? and data.Client[@props.Ktext] isnt undefined
                        return S
        getInitialState: () -> @reduce(Data) # a global Data
        componentDidMount: () -> @props.$button_el.click(@click)
        render: () ->
                if @props.Kable
                        able = @state.Able
                        able = !able if not (@props.Kable.indexOf('not') > -1) # That's a hack
                        @props.$button_el.prop('disabled', able)
                        @props.$button_el[if able then 'addClass' else 'removeClass']('disabled')
                @props.$button_el[if @state.Send then 'addClass' else 'removeClass']('active') if @props.Ksend?
                @props.$button_el.text(@state.Text) if @props.Ktext?
                return null
        click: (e) ->
                S = {}
                S[@props.Khide] = !@state.Hide if @state.Hide?  and @state.Hide # if the panel was hidden
                S[@props.Ksend] = !@state.Send if @props.Ksend? and @state.Send? # Q is a @state.Send? check excessive?
                S[@props.Ksig]  =  @props.Vsig if @props.Ksig?
                websocket.sendClient(S)
                e.stopPropagation() # preserves checkbox/radio
                e.preventDefault()  # checked/selected state
                return undefined

@TabsClass = React.createClass
        statics: component: (opt) -> React.renderComponent(TabsClass(opt), addNoscript(opt.$button_el))

        reduce: (data) ->
                if data?.Client?
                        S = {}
                        S.Hide = data.Client[@props.Khide] if                   data.Client[@props.Khide] isnt undefined # Khide is a required prop
                        S.Send = data.Client[@props.Ksend] if @props.Ksend? and data.Client[@props.Ksend] isnt undefined
                        return S
        getInitialState: () -> @reduce(Data) # a global Data
        componentDidMount: () ->
                @props.$button_el.click(@clicktab)
                @props.$hidebutton_el.click(@clickhide)
        render: () ->
                if @state.Hide
                        @props.$collapse_el.collapse('hide')
                        @props.$hidebutton_el.addClass('active')
                        return null
                @props.$hidebutton_el.removeClass('active')
                curtabid = +@state.Send # MUST be an int
                nots = @props.$collapse_el.not('[data-tabid="'+ curtabid + '"]')
                $(el).collapse('hide') for el in nots
                $(@props.$collapse_el.not(nots)).collapse('show')
                activeClass = (el) ->
                        xel = $(el)
                        tabid_attr = +xel.attr('data-tabid') # an int
                        xel[if tabid_attr == curtabid then 'addClass' else 'removeClass']('active')
                        return
                activeClass(el) for el in @props.$button_el
                return null
        clicktab: (e) ->
                S = {}
                S[@props.Ksend] = +$( $(e.target).attr('href') ).attr('data-tabid') # THIS. +string makes an int
                S[@props.Khide] = false if @state.Hide? and @state.Hide # if the panel was hidden
                websocket.sendClient(S)
                e.preventDefault()
                e.stopPropagation() # don't change checkbox/radio state
                return undefined
        clickhide: (e) ->
                (S = {})[@props.Khide] = !@state.Hide
                websocket.sendClient(S)
                e.stopPropagation() # preserves checkbox/radio
                e.preventDefault()  # checked/selected state
                return undefined

@RefreshInputClass = React.createClass
        statics: component: (opt) ->
                $ = opt.$; delete opt.$
                opt.$input_el = $.find('.refresh-input')
                opt.$group_el = $.find('.refresh-group')
                React.renderComponent(RefreshInputClass(opt), addNoscript(opt.$input_el))

        reduce: (data) ->
                if data?.Client? and (data.Client[@props.K]? or data.Client[@props.Kerror]?)
                        S = {}
                        S.Value = data.Client[@props.K]      if data.Client[@props.K]?
                        S.Error = data.Client[@props.Kerror] if data.Client[@props.Kerror]?
                        # console.log('newState', S)
                        return S
        getInitialState: () ->
                S = @reduce(Data) # a global Data
                # console.log('initialState', S)
                return S

        componentDidMount: () -> @props.$input_el.on('input', @submit)
        render: () ->
                # console.log('RefreshInputClass.render', @isMounted(), @state)
                @props.$input_el.prop('value', @state.Value) if @isMounted() and !@state.Error
                @props.$group_el[if @state.Error then 'addClass' else 'removeClass']('has-warning')
                return null
        submit: (e) ->
                (S = {})[@props.Ksig] = $(e.target).val()
                websocket.sendClient(S)
                e.preventDefault()
                e.stopPropagation() # don't change checkbox/radio state
                return undefined

@NewTextCLASS = (reduce) -> React.createClass
        newstate: (data) ->
                v = reduce(data)
                return {Text: v} if v?
        getInitialState: () -> @newstate(Data) # a global Data
        render: () -> React.DOM.span(null, @state.Text)

@setState = (obj, data) ->
        if data?
                delete data[key] for key of data when !data[key]?
                return obj.setState(data)

@update = () -> # currentClient
        return if (42 for param in location.search.substr(1).split('&') when param.split('=')[0] == 'still').length

        hideconfigmem = HideClass.component({key: 'HideconfigMEM', $collapse_el: $('#memconfig'), $button_el: $('header a[href="#mem"]'), reverseActive: true})
        hideconfigif  = HideClass.component({key: 'HideconfigIF',  $collapse_el: $('#ifconfig'),  $button_el: $('header a[href="#if"]'),  reverseActive: true})
        hideconfigcpu = HideClass.component({key: 'HideconfigCPU', $collapse_el: $('#cpuconfig'), $button_el: $('header a[href="#cpu"]'), reverseActive: true})
        hideconfigdf  = HideClass.component({key: 'HideconfigDF',  $collapse_el: $('#dfconfig'),  $button_el: $('header a[href="#df"]'),  reverseActive: true})
        hideconfigps  = HideClass.component({key: 'HideconfigPS',  $collapse_el: $('#psconfig'),  $button_el: $('header a[href="#ps"]'),  reverseActive: true})
        hideconfigvg  = HideClass.component({key: 'HideconfigVG',  $collapse_el: $('#vgconfig'),  $button_el: $('header a[href="#vg"]'),  reverseActive: true})

        hidemem = HideClass.component({key: 'HideMEM', $collapse_el: $('#mem'), $button_el: $('#memconfig').find('.hiding')})
        hidecpu = HideClass.component({key: 'HideCPU', $collapse_el: $('#cpu'), $button_el: $('#cpuconfig').find('.hiding')})
        hideps  = HideClass.component({key: 'HidePS',  $collapse_el: $('#ps'),  $button_el: $('#psconfig') .find('.hiding')})
        hidevg  = HideClass.component({key: 'HideVG',  $collapse_el: $('#vg'),  $button_el: $('#vgconfig') .find('.hiding')})

        ip       = React.renderComponent(NewTextCLASS((data) -> data?.Generic?.IP       )(), $('#generic-ip'      )   .get(0))
        hostname = React.renderComponent(NewTextCLASS((data) -> data?.Generic?.Hostname )(), $('#generic-hostname')   .get(0))
        uptime   = React.renderComponent(NewTextCLASS((data) -> data?.Generic?.Uptime   )(), $('#generic-uptime'  )   .get(0))
        la       = React.renderComponent(NewTextCLASS((data) -> data?.Generic?.LA       )(), $('#generic-la'      )   .get(0))

        iftitle  = React.renderComponent(NewTextCLASS((data) -> data?.Client?.TabTitleIF)(), $('header a[href="#if"]').get(0))
        dftitle  = React.renderComponent(NewTextCLASS((data) -> data?.Client?.TabTitleDF)(), $('header a[href="#df"]').get(0))

        psplus   = React.renderComponent(NewTextCLASS((data) -> data?.Client?.PSplusText)(), $('label.more[href="#psmore"]').get(0))
        psmore   = ButtonClass.component({Ksig: 'MorePsignal', Vsig: true,  Khide: 'HidePS', Kable: 'PSnotExpandable',  $button_el: $('label.more[href="#psmore"]')})
        psless   = ButtonClass.component({Ksig: 'MorePsignal', Vsig: false, Khide: 'HidePS', Kable: 'PSnotDecreasable', $button_el: $('label.less[href="#psless"]')})

        hideswap = ButtonClass.component({Khide: 'HideMEM', Ksend: 'HideSWAP', $button_el: $('label[href="#hideswap"]')})

        expandif = ButtonClass.component({Khide: 'HideIF',  Ksend: 'ExpandIF',  Ktext: 'ExpandtextIF',  Kable: 'ExpandableIF',  $button_el: $('label[href="#if"]')})
        expandcpu= ButtonClass.component({Khide: 'HideCPU', Ksend: 'ExpandCPU', Ktext: 'ExpandtextCPU', Kable: 'ExpandableCPU', $button_el: $('label[href="#cpu"]')})
        expanddf = ButtonClass.component({Khide: 'HideDF',  Ksend: 'ExpandDF',  Ktext: 'ExpandtextDF',  Kalbe: 'ExpandableDF',  $button_el: $('label[href="#df"]')})

        # NB buttons and collapses selected by class
        tabsif = TabsClass.component({Khide: 'HideIF', Ksend: 'TabIF', $collapse_el: $('.if-tab'), $button_el: $('.if-switch'), $hidebutton_el: $('#ifconfig').find('.hiding')})
        tabsdf = TabsClass.component({Khide: 'HideDF', Ksend: 'TabDF', $collapse_el: $('.df-tab'), $button_el: $('.df-switch'), $hidebutton_el: $('#dfconfig').find('.hiding')})

        refresh_mem = RefreshInputClass.component({K: 'RefreshMEM', Kerror: 'RefreshErrorMEM', Ksig: 'RefreshSignalMEM', $: $('#memconfig')})
        refresh_if  = RefreshInputClass.component({K: 'RefreshIF',  Kerror: 'RefreshErrorIF',  Ksig: 'RefreshSignalIF',  $: $('#ifconfig')})
        refresh_cpu = RefreshInputClass.component({K: 'RefreshCPU', Kerror: 'RefreshErrorCPU', Ksig: 'RefreshSignalCPU', $: $('#cpuconfig')})
        refresh_df  = RefreshInputClass.component({K: 'RefreshDF',  Kerror: 'RefreshErrorDF',  Ksig: 'RefreshSignalDF',  $: $('#dfconfig')})
        refresh_ps  = RefreshInputClass.component({K: 'RefreshPS',  Kerror: 'RefreshErrorPS',  Ksig: 'RefreshSignalPS',  $: $('#psconfig')})
        refresh_vg  = RefreshInputClass.component({K: 'RefreshVG',  Kerror: 'RefreshErrorVG',  Ksig: 'RefreshSignalVG',  $: $('#vgconfig')})

        memtable  = React.renderComponent(MEMtableCLASS(),  document.getElementById('mem'       +'-'+ 'table'))
        pstable   = React.renderComponent(PStableCLASS(),   document.getElementById('ps'        +'-'+ 'table'))
        dfbytes   = React.renderComponent(DFbytesCLASS(),   document.getElementById('dfbytes'   +'-'+ 'table'))
        dfinodes  = React.renderComponent(DFinodesCLASS(),  document.getElementById('dfinodes'  +'-'+ 'table'))
        cputable  = React.renderComponent(CPUtableCLASS(),  document.getElementById('cpu'       +'-'+ 'table'))
        ifbytes   = React.renderComponent(IFbytesCLASS(),   document.getElementById('ifbytes'   +'-'+ 'table'))
        iferrors  = React.renderComponent(IFerrorsCLASS(),  document.getElementById('iferrors'  +'-'+ 'table'))
        ifpackets = React.renderComponent(IFpacketsCLASS(), document.getElementById('ifpackets' +'-'+ 'table'))
        vgtable   = React.renderComponent(VGtableCLASS(),   document.getElementById('vg'        +'-'+ 'table'))

        onmessage = (event) ->
                data = JSON.parse(event.data)
                return if !data?

                console.log('DEBUG ERROR', data.Client.DebugError) if data.Client?.DebugError?
                if data.Reload? and data.Reload
                        window.setTimeout((() -> location.reload(true)), 5000)
                        window.setTimeout(websocket.close, 2000)
                        console.log('in 5s: location.reload(true)')
                        console.log('in 2s: websocket.close()')
                        return

                setState(pstable,  {PStable:  data.PStable,  PSlinks: data.PSlinks})
                setState(dfbytes,  {DFbytes:  data.DFbytes,  DFlinks: data.DFlinks})
                setState(dfinodes, {DFinodes: data.DFinodes, DFlinks: data.DFlinks})

                setState(hideconfigmem, hideconfigmem.reduce(data))
                setState(hideconfigif,  hideconfigif .reduce(data))
                setState(hideconfigcpu, hideconfigcpu.reduce(data))
                setState(hideconfigdf,  hideconfigdf .reduce(data))
                setState(hideconfigps,  hideconfigps .reduce(data))
                setState(hideconfigvg,  hideconfigvg .reduce(data))

                setState(hidemem,       hidemem      .reduce(data))
                setState(hidecpu,       hidecpu      .reduce(data))
                setState(hideps,        hideps       .reduce(data))
                setState(hidevg,        hidevg       .reduce(data))

                setState(ip,        ip      .newstate(data))
                setState(hostname,  hostname.newstate(data))
                setState(uptime,    uptime  .newstate(data))
                setState(la,        la      .newstate(data))

                setState(iftitle,   iftitle .newstate(data))
                setState(dftitle,   dftitle .newstate(data))

                setState(psplus,    psplus  .newstate(data))
                setState(psmore,    psmore  .reduce(data))
                setState(psless,    psless  .reduce(data))

                setState(hideswap,  hideswap.reduce(data))

                setState(expandif,  expandif.reduce(data))
                setState(expandcpu, expandcpu.reduce(data))
                setState(expanddf,  expanddf.reduce(data))

                setState(tabsif,    tabsif.reduce(data))
                setState(tabsdf,    tabsdf.reduce(data))

                setState(refresh_mem, refresh_mem.reduce(data))
                setState(refresh_if,  refresh_if .reduce(data))
                setState(refresh_cpu, refresh_cpu.reduce(data))
                setState(refresh_df,  refresh_df .reduce(data))
                setState(refresh_ps,  refresh_ps .reduce(data))
                setState(refresh_vg,  refresh_vg .reduce(data))

                setState(memtable,  data.MEM)
                setState(cputable,  data.CPU)
                setState(ifbytes,   data.IFbytes)
                setState(iferrors,  data.IFerrors)
                setState(ifpackets, data.IFpackets)
                setState(vgtable, {
                    VagrantMachines: data.VagrantMachines,
                    VagrantError:    data.VagrantError,
                    VagrantErrord:   data.VagrantErrord
                })

                console.log(JSON.stringify(data.Client), 'recvClient') if data.Client?

              # currentClient = React.addons.update(currentClient, {$merge: data.Client}) if data.Client?
              # data.Client = currentClient

                # update the tooltips
                $('span .tooltipable')    .popover({trigger: 'hover focus'})
                $('span .tooltipabledots').popover() # the clickable dots
                return

        @websocket = newwebsocket(onmessage)
        return

@ready = () ->
        (new Headroom(document.querySelector('nav'), {
                offset: 71 - 51
                # "relative" padding-top of the toprow
                # 71 is the absolute padding-top of the toprow
                # 51 is the height of the nav (50 +1px bottom border)
        })).init()

        $('.collapse').collapse({toggle: false}) # init collapsable objects

        $('span .tooltipable')      .popover({trigger: 'hover focus'})
        $('span .tooltipabledots')  .popover() # the clickable dots
        $('[data-toggle="popover"]').popover() # should be just #generic-hostname
        $('#generic-la')            .popover({
                trigger: 'hover focus',
                placement: 'right', # not 'auto right' until #generic-la is the last element for it's parent
                html: true, content: () -> $('#uptime').html()
        })

        $('body').on('click', (e) -> # hide the popovers on click outside
                $('span .tooltipabledots').each(() ->
                        # the 'is' for buttons that trigger popups
                        # the 'has' for icons within a button that triggers a popup
                        $(this).popover('hide') if !$(this).is(e.target) and $(this).has(e.target).length == 0 && $('.popover').has(e.target).length == 0
                        return)
                return)

        update() # (Data.Client)
        return

