const jwt = require('jsonwebtoken');

const secret = "fallback-secret-for-dev";
const token = jwt.sign({ user_id: "1", exp: Math.floor(Date.now() / 1000) + (60 * 60 * 24) }, secret);
console.log(token);
