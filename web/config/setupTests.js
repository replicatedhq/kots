import Enzyme from "enzyme";
import Adapter from "enzyme-adapter-react-16";
import "jest-enzyme";

class LocalStorageMock {
  constructor() {
    this.store = {};
  }

  clear() {
    this.store = {};
  }

  getItem(key) {
    return this.store[key] || null;
  }

  setItem(key, value) {
    this.store[key] = value.toString();
  }

  removeItem(key) {
    delete this.store[key];
  }
}

window.localStorage = new LocalStorageMock;

Enzyme.configure({ adapter: new Adapter() });
