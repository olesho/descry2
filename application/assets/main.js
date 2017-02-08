function ListPatterns() {
	$.get("/patterns", function(data){
		var list = data.split("\n");
		for (var i = 0; i < list.length; i++) {
			$("ul.patterns").append("<li><a href='/pattern/"+list[i]+"'>"+list[i]+"</a></li>")
		}  
	});
}

function ListProjects() {
	$.get("/projects", function(data){
		var list = data.split("\n");
		for (var i = 0; i < list.length; i++) {
			$("ul.projects").append("<li><a href='/project/"+list[i]+"'>"+list[i]+"</a></li>")
		}  
	});
}

function PutPattern(title, data) {
	$.ajax({
		url: "/pattern/"+title,
		type: "PUT",
		data: data
	}).done(function(resp){
		$(".data").html(resp);
	}).fail(function(resp){
		$(".data").html(resp);
	});
}

function TestPattern() {
	$.get("/assets/1478118249.html", function(data) {
		$.ajax({
			url: "/parse",
			type: "POST",
			data: JSON.stringify({origin: "https://www.atgstores.com/", data: data}),

		}).done(function(data) {
			$(".data").html(JSON.stringify(data));
		});
	})
}