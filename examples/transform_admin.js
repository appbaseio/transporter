module.exports = function(doc) {  
  if (doc.data.permission != "admin") {
  	doc.op = "n"
  }
  return doc;
}