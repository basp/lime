## Setup
Before we can create posts we first need to setup a working directory for our site. Create this directory somewhere where you can easily access it. Preferably with a nice short path:

    cd d:\temp
    mkdir mysite
    cd mysite
    lime

This will cause lime to spit out an error saying that it can't find the `d:\temp\mysite\_layouts` path so let's create that and try again:

    mkdir _layouts
    lime

Now we get a similar error but this time it's about the `d:\temp\mysite\_posts` path so let's create that too:

    mkdir _posts
    lime

Now we get no output at all which means that everything went as expected. 

## Posts
Except, nothing happened - when we look at the contents of the `mysite` directory we still only see the `_layouts` and `_posts` folders. This is because there is nothing to process yet! Let's fix that by creating a post. 

    cd _posts
    touch 2013-11-20-first-post.md

Take note of the filename. Post filenames have to follow that specific format or otherwise they will be ignored. Now edit that file with your favorite text editor to look like the following:

    ---
    title: First!
    ---
    # {{.page.title}}
    Yay for the first post!

The details will be explained later. Now if we run lime again it will actually do something for us. In this case it will create the `_site` output directory. Our post will be in that directory at `_site/2013/11/20/first-post.html`. 

## Layouts
The content of the post (below the second `---`) is what will be transformed to HTML by the Markdown parser. However, output alone will not turn it into a proper HTML page. For that, we need layouts. 

Layouts are (not surprisingly) placed in the `_layouts` directory and these are basically HTML files with template facilities. Let's create a new layout:

    cd _layouts
    touch post.html

This will create a layout file that we will use for our posts. Edit the file to look like the following:

    <!doctype html>
    <html>
    <head>
        <title>{{.page.title}}</title>
    </head>
    <body>
        {{.content}}
    </body>
    </html>

Note the template tags (`{{.page.title}}` and `{{.content}}` in this case). To use this template for our post we need to edit our `2013-11-20-first-post.md` file to look like below:

    ---
    title: First!
    layout: post
    ---
    # {{.page.title}}
    Yay for the first post!

Note the new `layout: post` declaration. This will tell lime to use the `post.html` template from the `_layouts` directory.