Source({name:"sourcetransformfile", namespace:"file.users"})
//Save all users
.save({name:"appbaseapp", namespace:"<APPNAME>.users"})

//Save just admin users
//.transform({filename: "./transform_admin.js", namespace:"."})
//.save({name:"appbaseapp", namespace:"<APPNAME>.admin"})

//Save just company names
//.transform({filename: "./transform_pickfields.js", namespace:"."})
//.save({name:"appbaseapp", namespace:"<APPNAME>.company"});
