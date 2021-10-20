/**
- Generate HNT wallet for donations
- Investigate using memcache for requests
- Set dates for the 2020/2021 tax year
- Deploy
*/

const INITIAL = 0;
const BAD_ADDRESS = 1;
const LOADING = 2;
const LOADED = 3;
const FAILED = 4;

function setUIState(state) {
  switch (state) {
    case INITIAL:
      $("#loading").hide();
      $("#loaded").hide();
      $("#error").hide();
      break;
    case BAD_ADDRESS:
      $(".input-group").addClass("has-error");
      break;
    case LOADING:
      $(".input-group").removeClass("has-error");
      $("#loading").show();
      $("#loaded").hide();
      $("#error").hide();
      break;
    case LOADED:
      $("#loading").hide();
      $("#loaded").show();
      $("#error").hide();
      break;
    case FAILED:
      $("#loading").hide();
      $("#loaded").hide();
      $("#error").show();
      break;
  }
}

function parseData(response) {
          // Generate CSV
        const header = "date, earnings, tokens mined, daily price\n";
        const csv = response
          .data
          .map((o) => {
            return o.date + "," + o.earnings + "," + o.tokens + "," + o.price + "\n";
          })
          .reduce((sum, value) => sum + value);

        $("#csv-results").text(header + csv);

        // Calculate total value
        const totalEarnings = response
          .data
          .map((x) => x.earnings)
          .reduce((sum, value) => sum + value);

        $("#total-value").text(formatter.format(totalEarnings));
}

function pollForData(hntAddress, taxYear) {
   $.getJSON("/data/" + hntAddress + "?tax_year=" + taxYear)
      .done(function(response) {
        console.log(response);
        if (!response.data) {
          setTimeout(() => pollForData(hntAddress, taxYear), 10000);
          return;
        }
     
        setUIState(LOADED);
        parseData(response)
      })
      .fail(function(data) {
        setUIState(FAILED);
      });
}

const formatter = new Intl.NumberFormat('en-GB', {
  style: 'currency',
  currency: 'GBP'
});

$(function() {
  setUIState(INITIAL);
  
  $("#show-csv").click(() => {
    $("#csv-results").toggle();
  })

  $("#submitBtn").click(() => {
    const taxYear = $("input[name=tax-year]:checked").val();
    const hntAddress = $("input#address").val();

    if (!hntAddress || hntAddress.length != 51) {
      setUIState(BAD_ADDRESS);
      return;
    }

    setUIState(LOADING);
    
    // enqueue the request
    $.getJSON("/enqueue/" + hntAddress + "?tax_year=" + taxYear)
      .done(function(response) {
         pollForData(hntAddress, taxYear)
    });  
  });
})
