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

However, we can do even better by using a master template. Create a new file in the `_layouts` directory:

    touch master.html

And modify it to look like below (you can copy the `post.html` template contents for this):

    <!doctype html>
    <html>
    <head>
        <title>{{.page.title}}</title>
    </head>
    <body>
        {{.content}}
    </body>
    </html>

Now edit the `post.html` template:

    ---
    layout: master
    ---
    <div class="post">
        {{.content}}
    </div>

Note how we added a reference to the `master.html` layout with `layout: master`. Now this layout will be nested into the master. If we run lime again we'll se that everything will be nested and output to the `_site` folder just like before.

The benefit of using nested layouts is that we can create an even look and feel for all our various posts and pages without the need to repeat ourselves.

## Pages
Besides posts (which are somewhat special) we can also add ordinary HTML pages to our site. These don't have a special directory but are just fetched from the site working directory (`mysite` in this case). The only requirement for these files is that they start with a `---` line. Let's create an index file for our site. Place this file directly below the `mysite` directory:

    touch index.html

And modify it so it looks like this:

    ---
    layout: master
    title: Home
    ---
    <h1>{{.page.title}}</h1>
    <p>This is the index page</p>

If we now run lime again you'll see that `index.html` is placed into the `_site` output directory. As shown above, page files can use the same template and layout facilities as posts.

## Other files
You can even place other files below the `mysite` directory and have them processed by lime. Similar to page files, the only requirement is that they start with a `---` line. 

We could, for example, include a CSS file too:

    ---
    body { font-family: sans-serif; }

And this will be picked up by lime and put into the `_site` output directory. Of course, you can still use layouts and other metadata facilities too if you want:

    ---
    defaultColor: #acacac
    ---
    body {
        color: {{.defaultColor}}
    }

This might be more useful for some files than others but the facilities are there if you want to use them.