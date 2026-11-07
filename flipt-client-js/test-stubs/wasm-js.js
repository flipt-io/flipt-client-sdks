// Stand-in for the generated wasm-bindgen glue so src/ can be unit tested
// without building flipt-engine-wasm-js.
export default async function init() {}

export class Engine {
  constructor() {
    this.snapshots = [];
  }

  // Like the real engine, reports whether the state changed.
  snapshot(data) {
    const prev = this.snapshots[this.snapshots.length - 1];
    this.snapshots.push(data);
    return JSON.stringify(prev) !== JSON.stringify(data);
  }

  get_snapshot() {
    return '';
  }
}
