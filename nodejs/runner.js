'use strict';

console.oldLog = console.log;
console.log = function(value)
{
    console.oldLog(value);
    return value;
};

const inputArgs = process.argv.slice(2);
let file;

if (inputArgs[0] === '--input')
    file = inputArgs[1]

const fs = require('fs')

try {
    const data = fs.readFileSync(file, 'utf8')
    const inputJson = JSON.parse(data)
    const inputCode = inputJson.code

    eval(inputCode)

} catch (err) {
    console.error(err)
}
