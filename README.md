## tachyon

[![Build Status](https://travis-ci.org/vektra/tachyon.svg?branch=master)](https://travis-ci.org/vektra/tachyon)

Tachyon is an experimental configuration management tool inspired by ansible implemented in golang.

#### Ok.. why?

I find the best way to learn something is to try to implement it.
I'm curious about ansible's model for configuration management and
as a fun weekend project began I this project.

#### Is this usable?

If you need to run some yaml that executes commands via shell/command, sure!
Otherwise no. I'll probably continue to play with it, adding more functionality
and fleshing out some ideas I've got.

#### Oohh what ideas?

* Exploit golang's single binary module to bootstrap machines and run plays remotely.
* Use golang's concurrency to make management of large scale changes easy.
* Use github.com/evanphx/ssh to do integrated ssh
* Allow creation of modules via templated tasks

#### Is that a lisp directory I see?

It is! ansible uses python as it's implementation lang and thus also uses it as
it's runtime eval language. Obviously I can't do that and I don't wish to runtime
eval any golang code. Thus I have opted to embed a simple lisp intepreter
(taken and modified from [http://github.com/janne/go-lisp](github.com/janne/go-lisp))
to run code. For instance:

```yaml
name: Tell everyone things are great
action: shell echo wooooo!
when: $(== everything "awesome")
```

#### What should I do with this?

Whatever you want. Play around, tell me what you think about it. Send PRs for crazy ass
features!
