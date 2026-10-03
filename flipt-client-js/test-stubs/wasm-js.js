// Stand-in for the generated wasm-bindgen glue so src/ can be unit tested
// without building flipt-engine-wasm-js.
export default async function init() {}

export class Engine {
  constructor() {
    this.snapshots = [];
  }

  snapshot(data) {
    this.snapshots.push(data);
  }

  get_snapshot() {
    return '';
  }
}
