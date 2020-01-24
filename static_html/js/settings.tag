<settings>
  <h2>{ opts.title }</h2>

	<div class="row">
    <form class="col s12" name="settings-form" id="settings-form">
      <div class="row">
        <div class="input-field col s12">
          <input value="{opts.location_confidence}" id="location_confidence" type="number" min="1">
          <label for="location_confidence" class="active">Location Confidence<i class="material-icons tooltipped" data-position="right" data-tooltip="How many times in a row a beacon should be seen in a location before it's considered located there. The higher this value is, the more accuracy is improved, but the longer it will need to take effect">help</i></label>
        </div>
      </div>
			<div class="row">
				<div class="input-field col s12">
					<input value="{opts.last_seen_threshold}" id="last_seen_threshold" type="number" min="1">
					<label for="last_seen_threshold" class="active">Last Seen Threshold<i class="material-icons tooltipped" data-position="right" data-tooltip="How many seconds until a beacon is considered gone">help</i></label>
				</div>
			</div>
      <div class="row">
        <div class="input-field col s12">
          <input value="{opts.beacon_metrics_size}" id="beacon_metrics_size" type="number" min="1">
          <label for="beacon_metrics_size" class="active">Beacon metrics size<i class="material-icons tooltipped" data-position="right" data-tooltip="How many previous signal readings, from all detectors, to keep for a beacon for calculation">help</i></label>
        </div>
      </div>
      <div class="row">
        <div class="input-field col s12">
          <input checked="{opts.enforce_rssi_threshold}" id="enforce_rssi_threshold" type="checkbox">
          <label for="enforce_rssi_threshold" class="active">Enforce the RSSI threshold.<i class="material-icons tooltipped" data-position="right" data-tooltip="Ignore beacon transmissions below the RSSI minumum threshold below">help</i></label>
        </div>
      </div>
      <div class="row">
        <div class="input-field col s12">
          <input value="{opts.rssi_min_threshold}" id="rssi_min_threshold" type="number">
          <label for="rssi_min_threshold" class="active">RSSI minimum threshold<i class="material-icons tooltipped" data-position="right" data-tooltip="Minimum RSSI (in dB) to include in beacon positioning calculations">help</i></label>
        </div>
      </div>
			<h4>Home Assistant</h4>
      <div class="row">
        <div class="input-field col s12">
          <input value="{opts.ha_send_interval}" id="ha_send_interval" type="number" min="1">
          <label for="ha_send_interval" class="active">Home Assistant Publish Interval<i class="material-icons tooltipped" data-position="right" data-tooltip="How often to send beacon info to Home Assistant, setting this too low can cause high CPU usage on Raspberry Pi">help</i></label>
        </div>
      </div>
      <div class="row">
        <div class="input-field col s12">
          <input checked="{opts.ha_send_changes_only}" id="ha_send_changes_only" type="checkbox">
          <label for="ha_send_changes_only" class="active">Only Send Changes in Location to Home Assistant<i class="material-icons tooltipped" data-position="right" data-tooltip="Only send changes to Home Assistant when the location of a beacon changes">help</i></label>
        </div>
      </div>
      <div class="row">
        <div class="input-field col s12">
          <input checked="{opts.debug}" id="debug" type="checkbox">
          <label for="debug" class="active">Write console debug log</label>
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
		this.on('mount', function() {

			$('.tooltipped').tooltip({delay: 20});

			$.ajax(
			{
				url: "api/settings",
				type: "GET",
				dataType: 'json',
			})
			.done(function(data) {
				if(data.location_confidence) {
					$("#location_confidence").val(data.location_confidence);
				}
				if(data.last_seen_threshold) {
					$("#last_seen_threshold").val(data.last_seen_threshold);
				}
				if(data.beacon_metrics_size) {
					$("#beacon_metrics_size").val(data.beacon_metrics_size);
				}
				if(data.rssi_min_threshold) {
					$("#rssi_min_threshold").val(data.rssi_min_threshold);
				}
				if(data.ha_send_interval) {
					$("#ha_send_interval").val(data.ha_send_interval);
				}
				if(data.ha_send_changes_only) {
					$("#ha_send_changes_only").prop("checked", data.ha_send_changes_only);
				}
				if(data.enforce_rssi_threshold) {
					$("#enforce_rssi_threshold").prop("checked", data.enforce_rssi_threshold);
                }
				if(data.debug) {
					$("#debug").prop("checked", data.debug);
                }
                $("#enforce_rssi_threshold").trigger("change");
            });

            $("#enforce_rssi_threshold").on("change", function(e) {
				$("#rssi_min_threshold").prop("disabled", !($("#enforce_rssi_threshold").prop("checked")));
				$("#rssi_min_threshold").prop("readonly", !($("#enforce_rssi_threshold").prop("checked")));
            });

			$("#settings-form").submit(function(event){
				event.preventDefault();
				var form_data = {}
				form_data["location_confidence"] = parseInt($("#location_confidence").val());
				form_data["last_seen_threshold"] = parseInt($("#last_seen_threshold").val());
				form_data["beacon_metrics_size"] = parseInt($("#beacon_metrics_size").val());
				form_data["rssi_min_threshold"] = parseInt($("#rssi_min_threshold").val());
				form_data["ha_send_interval"] = parseInt($("#ha_send_interval").val());
				form_data["ha_send_changes_only"] = $("#ha_send_changes_only").prop("checked");
				form_data["enforce_rssi_threshold"] = $("#enforce_rssi_threshold").prop("checked");
				form_data["debug"] = $("#debug").prop("checked");
				//console.log(form_data);
				$.ajax(
				{
					url: "api/settings",
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

</settings>

