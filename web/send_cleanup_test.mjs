import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import vm from "node:vm";

// Exercise the production function with controlled failures at each await.
const source = readFileSync(new URL("./app.js", import.meta.url), "utf8");
const sendSource = source.slice(source.indexOf("async function sendStream("),
  source.indexOf('\n$("send-btn").onclick'));

for (const failure of ["none", "dial", "file", "write", "closeWrite", "read"]) {
  test(`send cleanup: ${failure}`, async () => {
    let closes = 0;
    const error = new Error(`injected ${failure} failure`);
    const failAt = (step) => { if (step === failure) throw error; };
    const state = {};
    const conn = {
      write: async () => failAt("write"),
      closeWrite: async () => failAt("closeWrite"),
      read: async () => { failAt("read"); return null; },
      close: () => { closes++; },
    };
    const send = vm.runInNewContext(sendSource + "\nsendStream", {
      CHUNK: 65536, derpMapURL: "http://unused.test", verbose: false,
      window: { tcTest: state },
      tailcatDial: async () => { failAt("dial"); return conn; },
    });
    const pending = send("test", 1, async () => {
      failAt("file"); return new Uint8Array(1);
    }, { textContent: "" });
    if (failure === "none") {
      await pending;
      assert.equal(state.sendDone, true);
      assert.equal(state.sentBytes, 1);
    } else {
      await assert.rejects(pending, e => e === error);
      assert.equal(state.sendDone, undefined);
    }
    assert.equal(closes, failure === "dial" ? 0 : 1);
  });
}
