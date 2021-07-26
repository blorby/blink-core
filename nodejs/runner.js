'use strict';

console.oldLog = console.log;
console.log = function(value)
{
    console.oldLog(value);
    return value;
};

const inputArgs = process.argv.slice(2);
let data;

if (inputArgs[0] === '--input')
    data = inputArgs[1]

const inputBuffer = new Buffer.from(data, 'base64');
const inputData = inputBuffer.toString('utf-8');
const inputJson = JSON.parse(inputData)
const codeBuffer = new Buffer.from(inputJson.code, 'base64');
const inputCode = codeBuffer.toString('utf-8');

eval(inputCode)
