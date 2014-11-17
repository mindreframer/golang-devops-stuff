# contributing to RedSkull

This is a short guide on contributing to RedSkull. It will be fleshed out as
lessons are learned.

# Basic Process

From a functional standpoint this is on Github so clone the repository, create
a dedicated branch, make changes, merge from upstream, validate code,
submit PR, and finally stop making changes unless needed to address issues,
questions, concerns, etc..

## Git Setup

A good way to ensure your PR is against the current master repo when
submitted, you can do the following:

Add the following alias to your .gitconfig file:

```
[alias]
  pu = !"git fetch origin -v; git fetch upstream -v; git merge
  upstream/master"
```

Then run the following command to set upstream:

`git remote add upstream git@github.com:TheRealBill/redskull.git` if using
SSH, or `git remote add upstream
https://github.com/therealbill/redskull.git` if using HTTPS to
login.

Now, for the sync step you simply run `git pu` from within the repo
and upstream changes will be pulled and merged - provided there are no
conflicts.

## Making Your Changes

Before making *any* changes create a dedicated branch with a good
succinct name. If it fixes an issue in Github, call it something like
'bugfix-ghNNN" where NNN is the issue number. If it fixes something
not in the issue list, create the issue first thus reducing to the
previously known solution.

For new features, see above but a scheme such as 'feature-ghNNN' would
be ideal.

This way it will help prioritize merges.

If you're makign documentation improvements, first woohoo!. Next see
above but perhaps something like 'docs-ghNNN'?

Why this scheme? Because GitHub's issues interface is a bit dodgy and
I think this might help with some automatic filtering - and should
help me when I look at your PR to put me in the right context as I dig
into it.

And just in case it wasn't clear above - one issue per PR, please. I'd
rather have a quick review of 5 PRs than one big one.


## Validating Code

In addition to unit testing where it makes sense, you will need to run
some go checks prior to submitting code. This is a good way to do
them:

```shell
go fmt
go vet
go build
```
Once it builds, run it and test the functions touched by your changes.
If it breaks, fix it.

This is considered the minimum you should be doing to ensure your code
doesn't break upstream.

## Submitting the Pull Request

First things first, squash your commits. If you're like me you
probably commit early and commit often. This is great as you go along,
but not so great for me as I sift through them. So, please squash your
commits. Don't know what that means? Yeah neither did I until someone
made this request for one of my PRs somewhere.

To learn how, I recommend this page: http://bit.ly/10FV2x7

Frankly if the PR only has a few (i.e under 6-ish) I'll probably let it
be. Unless Red Skull gets flooded with PRs in which case I'll likely
have to be a bit more ... adamant about it.

Detail your changes, reference any Github issues related and any
enhancement proposals, and explain what your PR is all about. No
matter how trivial your change is. Pull Requests without details will
be summarily closed, occasionally with a comment indicating why.

After you make issue the PR do not make changes in that branch. Many
people coming from the Subversion world don't realize additional
changes are automatically added to the pull request. Yeah, that bit me
too. So don't do it. Besides, if you're requesting that dedicated
branch be merged aren't you done with it anyway? 

Of course, if changes have to be made to address issues or for
improvements during the PR review process by all means make them.

# Licensing of Pull Request Code

I find this to be a thorny and tricky subject, but IMO it is best to
address it early. Basic rule here is for non-trivial code to submit it
under the Apache v2 license, public domain, or assign me copyright.
Why? To prevent problems laer down the road. If it is a big chunk of
code please try to note in the PR your options here. I'm not going to
be a stickler about it u tit would help prevent issues down the road.

If you don't specify it will be assumed you are submitting it under
the Apache license as noted above. I expect this to be enough, really.
But lately some fairly hgh profile projects have been hit by this.


# Summary

If you are a seasoned contrinutor to other projects on Github much of
this will be familiar to you and kudos for reading this far anyway! If
not, well now you've been given a quick lesson in how to get your
contributions' acceptance chance improved for many projects.



