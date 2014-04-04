# -*- python -*-
import os, os.path, shlex
import SCons

Default(None)

Files = lambda ROOT: [os.path.join(sub, f) for sub, _, fs in os.walk(ROOT) for f in fs]
def bindata(target, source, env, for_signature):
    fix = source[0].path         if isinstance(source[0], SCons.Node.FS.Dir) else  os.path.dirname(source[0].path)
    src = source[0].path +'/...' if isinstance(source[0], SCons.Node.FS.Dir) else  os.path.dirname(source[0].path)
    return ' '.join(shlex.split('''
go-bindata
  -pkg    {pkg}
  -o      {o}
  -tags   {tags}
  -prefix {prefix}
          {source}
'''.format(
    o   = target[0],
    pkg = os.path.basename(
        os.path.dirname( # sorry about that
            os.path.abspath(
                target[0].path))),
    prefix = fix,
    source = src,
    tags   = env['TFLAGS'],
)))

env = Environment(ENV={'PATH': os.environ['PATH'],
                       'HOME': os.path.expanduser('~')}, BUILDERS={
    'bindata': Builder(generator=bindata),
    'sass':    Builder(action='sass $SOURCES $TARGETS'),
})

assets = (Dir('assets/'),             Files('assets/'))
mintpl = ('templates.min/index.html', Files('templates.min/'))
Default(env.Clone(TFLAGS= 'production')       .bindata('src/ostential/view/bindata.production.go',   source=mintpl))
Default(env.Clone(TFLAGS='!production -debug').bindata('src/ostential/view/bindata.devel.go',        source=mintpl))
Default(env.Clone(TFLAGS= 'production')       .bindata('src/ostential/assets/bindata.production.go', source=assets))
Default(env.Clone(TFLAGS='!production -debug').bindata('src/ostential/assets/bindata.devel.go',      source=assets))

Default(env.sass('assets/css/index.css', 'style/index.scss'))
