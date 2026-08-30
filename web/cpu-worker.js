const { parentPort, workerData } = require("worker_threads");

const ms = Number(workerData && workerData.ms) || 200;
const end = Date.now() + ms;
let x = 0;
while (Date.now() < end) {
  x = Math.sqrt(x + Math.random());
}
parentPort.postMessage(x);
