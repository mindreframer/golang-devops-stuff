# Documentation

Here there be dragons, and soon the docs.


# Documentation Styles and Guidelines

## Format

The preferred format of all documentation efforts is markdown. 

## Doc Types

There two primary documentation types in the repo. First, we have
per-file documentation. This is a technique I've found quite useful.
Under this approach each .go file has an accopmanying .md file where
it would be useful.  For example a constellation.go could have a
constellation.md file explaining more details about the
constellation.go code and it's use.  


Naturally code-specific documentation should be in the immediate code
via comments.  However, sometimes you want more details on why a
particular code approach was used or more details on the use of the
code in the file.  This is what should go into the .md

In the docs directory should go documentation on a broader scope.
Design documents, interface documents, API docs, usage guidelines,
etc. are all items which should be documented here in the docs
directory. Additionally tutorials should be in the docs/tutorials
tree.



