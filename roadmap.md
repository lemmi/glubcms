# Roadmap

## Approximate order for features

1. Draft directory structure
2. Build the server with ability to **render git trees**
	- HOST/[commitid/]level0/.../levelN[.rss|/file]
	- one menu per level
	- use PATH+commit id as key for caching
	- use optional *commit id* as *"time machine"*
	- basic templates (html, rss) and styles via css (desktop, mobile)
3. Add **authentication** to restrict access to some paths 
4. Draft json-api to add new articles
5. Add functionality to add new articles, the plan:
	- every **new edit** takes place in **new branch**
	- after edit, **merge** as far **back** as possible (for some levels (master branch) higher privileges should be required)
	- **sign-off mechanic** (merge branch further by higher privileged user)
6. Add markdown editor
7. File upload

## Other Goals

- project page
- once the server builds on master, no new commit should break building
- demonstrations, benchmarks
- start documentation after adding articles is possible
- **only-one-binary-required-installation**
- reasonable *performance* for public pages even on *raspberry pi*
- target OS is linux, but retain ability to run on as many plattforms as possible
