# descry
Golang package and server for mapping and retrieving certain fields from HTML to JSON. 
It makes parsing, crawling and scraping different and related sources an easy task.

### Server API ###

**GET /patterns**
Get the list of all available patterns

**PUT /pattern/{path:/source/subsource/.../PatternName.xml}**
Creates XML pattern on server

**DELETE /pattern/{path:/source/subsource/.../PatternName.xml}**
Deletes XML pattern or folder

**GET /pattern/{path:/source/subsource/.../PatternName.xml}**
Get XML pattern from server

**GET /reload**
Reloads all available patterns to memory. Uploaded pattern is only active after server reload.

**POST /parse**
Parse given HTML file from given source.
Request body is expected to be HTML.
You should set an origin URL in HTTP header "X-Source".

### Example ###

Suppose you got http://www.first-source.com/pages/example.html containing list of URLs 
linked to http://www.second-source.io/blog/... And your task is to get crawl then parse data from last source.

We need to write two XML patterns:

FirstSourceLink.xml
```
<Pattern title="FirstSource">
	<URLRules>
    <!-- we set up Regex rule to include only URLs we need -->
		<RegexInclude>^http://www.first-source.com/pages/example.html$</RegexInclude>
	</URLRules>
	<Field title="Link">
		<XPath>//ul/li[@class='list-item']/a/@href</XPath>
		<!-- Type []string here means we expect to get list of strings -->
    	<Type name="string[]" />
	</Field>
</Pattern>
```
SecondSourcePost.xml
```
<Pattern title="SecondSource">
	<URLRules>
		<RegexInclude>^http://www.second-source.com/blog/[a-z0-9-_]+$</RegexInclude>
	</URLRules>
	<Field title="BlogPost">
  
		<XPath>//div[@class='blog-post']</XPath>
    <!-- Type 'struct' here means we expect to get a structure containing different type fields -->
		<Type name="struct" />
    
    <!-- Our sub-field named Title -->
    <Field title="Title">
      <XPath>//h1</XPath>
      <Type name="string" />
    </Field>
	</Field>
</Pattern>
```

Now we need to install patterns on the server (remember to use PUT request):

```
curl -X PUT -d '... Put First pattern XML data here ....' http://localhost:5000/pattern/first-source.com/FirstSourceLink.xml
curl -X PUT -d '... Put Second pattern XML data here ....' http://localhost:5000/pattern/second-source.io/SecondSourcePost.xml
```
After installing patterns server is ready for use. Download page from the remote source and let Descry recognize it:

```
curl -X POST -d '... Put raw HTML here ...'  http://localhost:5000/parse -H "X-Source: http://www.first-source.com/pages/example.html"
```
HTML source to be sent with this request:

```
<html>
	<head></head>
	<body>
		<ul>
			<li class="list-item"><a href="http://www.second-source.com/blog/Hello_Word"></a></li>
			<li class="list-item"><a href="http://www.second-source.com/blog/First_Blog_Post"></a></li>
			<li class="list-item"><a href="http://www.second-source.com/blog/Got_something_for_ya"></a></li>
		</ul>
	</body>
</html>
```

Responcs would be in JSON:
```
{
  "first-source.com": {
    "FirstSourceLink.xml": {
      "Link":[
        "http://www.second-source.com/blog/Hello_Word",
        "http://www.second-source.com/blog/First_Blog_Post",
        "http://www.second-source.com/blog/Got_something_for_ya"
       ]
     }
   }
 }
