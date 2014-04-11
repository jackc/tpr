(function() {
  "use strict";

  window.promiseFailed = function(e) {
    console.log("Promise failed");
    console.log(e.toString());
    console.log(e.stack);
  };
})();
