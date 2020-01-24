<add-beacon>
  <h2>{ opts.title }</h2>

	 <div class="row">
    <form class="col s12" name="beacon-form" id="beacon-form">
      <div class="row">
        <div class="input-field col s12">
          <input disabled value="{opts.beacon_id}" id="beacon_id" type="text">
          <label for="beacon_id" class="active">Beacon ID</label>
        </div>
      </div>
      <div class="row">
        <div class="input-field col s12">
          <input id="beacon_name" value="{decodeURIComponent(opts.beacon_name)}" minlength=1 type="text" class="validate">
          <label for="beacon_name" class="active" data-error="wrong" data-success="ok">Beacon name</label>
        </div>
      </div>
      <div class="row">
        <div class="input-field col s12">
          <input id="beacon_enabled" value="{opts.beacon_enabled}" minlength=1 type="checkbox" class="validate">
          <label for="beacon_enabled" class="active" data-error="wrong" data-success="ok">Enabled</label>
        </div>
      </div>
      <div class="row">
				<button class="btn waves-effect waves-light" type="submit" name="action">Submit
		  		<i class="material-icons right">send</i>
		   </button> 
      </div>
    </form>
  </div>

	<script>
		this.on('mount', function(){
			console.log(opts);
			console.log(opts.beacon_enabled);
			if(opts.beacon_enabled == "true") {
				$("#beacon_enabled").prop("checked", true);
			}
			$("#beacon-form").submit(function(event){
				event.preventDefault();
				var form_data = {}
				form_data["beacon_id"] = $("#beacon_id").val();
				form_data["name"] = $("#beacon_name").val();
				form_data["beacon_enabled"] = $("#beacon_enabled").prop("checked");
				//console.log(form_data);
				$.ajax(
				{
					url: "api/beacons",
					type: "POST",
					contentType: 'application/json; charset=UTF-8',
					data: JSON.stringify(form_data),
				})
				.done(function(data) {
					window.location.hash = '#home';
				});
			});
		})
	</script>

</add-beacon>

