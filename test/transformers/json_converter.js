module.exports = function(doc) {
  console.log("transformer: " + doc.data)
  var jsonData = {
    "data": doc.data
  }
  doc.data = jsonData
  return doc
}
