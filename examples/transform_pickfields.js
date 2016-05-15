module.exports = function(doc) {  
  doc.data = _.pick(doc.data, ["email", "company", "phone"]);
  return doc;
}
