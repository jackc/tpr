var monthNames = ["January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"];
var ordinalSuffix = function(n) {
  n = n % 100;
  if (11 <= n && n < 13) {
    return "th";
  }

  n = n % 10;
  if (n == 1) return "st";
  if (n == 2) return "nd";
  if (n == 3) return "rd";
  return "th";
};

var hour12 = function(h) {
  h = (h % 12);
  return h == 0 ? 12 : h;
}

var min = function(m) {
  return m < 10 ? "0" + m : m;
}

var xm = function(h) {
  return h < 12 ? "am" : "pm"
}

var toTPRString = function(date) {
  var t = date,
      y = t.getFullYear(),
      m = monthNames[t.getMonth()],
      d = t.getDate(),
      o = ordinalSuffix(d),
      h = t.getHours(),
      mm = t.getMinutes();

  return m + " " + d + o + ", " + y + " at " + hour12(h) + ":" + min(mm) + " " + xm(h);
}

export {toTPRString}
