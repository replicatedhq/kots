import { expect } from 'chai';
import { it } from 'mocha';
import { appendVersion } from '../publisher';

describe ('appendVersion', () => {
  it('appends a new version', () => {
    const versions = appendVersion([{version: '1.84.0'}, {version: '1.85.0'}], {version: '1.84.1'});
    expect(versions).to.deep.equal([{version: '1.85.0'}, {version: '1.84.1'}, {version: '1.84.0'}]);
  });

  it('de-dupes duplicate versions', () => {
    const versions = appendVersion([{version: '1.84.0'}, {version: '1.85.0'}], {version: '1.85.0'});
    expect(versions).to.deep.equal([{version: '1.85.0'}, {version: '1.84.0'}]);
  });
});
