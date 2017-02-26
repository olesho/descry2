if ($(".pattern-checking").length) {
  $(".pattern").bind('input propertychange', function() {
    $.ajax({
      url: "/check",
      type: "POST",
      data: $(this).val()
    }).done(function(resp){
     // var resp = JSON.parse(json);
      if (resp.Message) {
        $(".alert").removeClass("hidden");
        $(".alert").html(resp.Message);
      } else {
        $(".alert").addClass("hidden");
        $(".alert").empty();
      }

      const formatter = new JSONFormatter(resp.Payload);
      $(".result").html(formatter.render());
      formatter.openAtDepth(Infinity);
      //$(".results").html(JSON.stringify(resp.Payload));        
    }).fail(function(resp){
      $(".alert").removeClass("hidden");
      $(".alert").html(resp);
    });
  });
}