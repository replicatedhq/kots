/**
 * Test file for cmx-versions functions
 * 
 * Before running this test, make sure to install the required dependencies:
 * cd .github/actions/cmx-versions
 * npm install semver
 * 
 * Then run the test from the repository root:
 * node .github/actions/cmx-versions/test.js
 * 
 * Make sure you're in the root directory of the repository when running the test.
 */

try {
    var semverCoerce = require('semver/functions/coerce');
    var semverMajor = require('semver/functions/major');
    var semverMinor = require('semver/functions/minor');
    var semverGt = require('semver/functions/gt');
    var semverRSort = require('semver/functions/rsort');
} catch (error) {
    console.error('Error: The semver package is not installed.');
    console.error('Please run the following commands to install it:');
    console.error('cd .github/actions/cmx-versions');
    console.error('npm install semver');
    console.error('\nThen run this test again.');
    process.exit(1);
}

const assert = require('assert');
const path = require('path');
const fs = require('fs');

// Import the functions from index.js
// We need to disable the actual execution of getClusterVersions
const indexPath = path.join(__dirname, 'index.js');

// Create a mock for @actions/core to prevent errors when running the test
const coreMock = {
    getInput: () => null,
    setOutput: () => {},
    setFailed: () => {}
};

// Read the index.js file content
const indexContent = fs.readFileSync(indexPath, 'utf8');

// Extract the function definitions we need
const getLatestVersionFn = extractFunction(indexContent, 'getLatestVersion');
const getLastMajorMinorVersionsFn = extractFunction(indexContent, 'getLastMajorMinorVersions');
const sortVersionsFn = extractFunction(indexContent, 'sortVersions');
const getLatestMinorVersionsFn = extractFunction(indexContent, 'getLatestMinorVersions');

// Create functions from the extracted code
const getLatestVersion = eval(`(${getLatestVersionFn})`);
const sortVersions = eval(`(${sortVersionsFn})`);
const getLastMajorMinorVersions = eval(`(${getLastMajorMinorVersionsFn})`);
const getLatestMinorVersions = eval(`(${getLatestMinorVersionsFn})`);


// Helper function to extract a function from a file content
function extractFunction(content, functionName) {
    const functionRegex = new RegExp(`function\\s+${functionName}\\s*\\([^)]*\\)\\s*{[\\s\\S]*?\\n}`);
    const match = content.match(functionRegex);
    if (match) {
        return match[0];
    }
    throw new Error(`Function ${functionName} not found in index.js`);
}

// Mock API response data
const apiResponse = {
    "cluster-versions": [
        {
            "short_name": "k3s",
            "versions": [
                "1.24.1", "1.24.2", "1.24.3", "1.24.4", "1.24.6", "1.24.7", "1.24.8", "1.24.9", "1.24.10",
                "1.24.11", "1.24.12", "1.24.13", "1.24.14", "1.24.15", "1.24.16", "1.24.17",
                "1.25.0", "1.25.2", "1.25.3", "1.25.4", "1.25.5", "1.25.6", "1.25.7", "1.25.8", "1.25.9",
                "1.25.10", "1.25.11", "1.25.12", "1.25.13", "1.25.14", "1.25.15", "1.25.16",
                "1.26.0", "1.26.1", "1.26.2", "1.26.3", "1.26.4", "1.26.5", "1.26.6", "1.26.7", "1.26.8",
                "1.26.9", "1.26.10", "1.26.11", "1.26.12", "1.26.13", "1.26.14", "1.26.15",
                "1.27.1", "1.27.2", "1.27.3", "1.27.4", "1.27.5", "1.27.6", "1.27.7", "1.27.8", "1.27.9",
                "1.27.10", "1.27.11", "1.27.12", "1.27.13", "1.27.14", "1.27.15", "1.27.16",
                "1.28.1", "1.28.2", "1.28.3", "1.28.4", "1.28.5", "1.28.6", "1.28.7", "1.28.8", "1.28.9",
                "1.28.10", "1.28.11", "1.28.12", "1.28.13", "1.28.14", "1.28.15",
                "1.29.0", "1.29.1", "1.29.2", "1.29.3", "1.29.4", "1.29.5", "1.29.6", "1.29.7", "1.29.8",
                "1.29.9", "1.29.10", "1.29.11", "1.29.12", "1.29.13", "1.29.14",
                "1.30.0", "1.30.1", "1.30.2", "1.30.3", "1.30.4", "1.30.5", "1.30.6", "1.30.7", "1.30.8",
                "1.30.9", "1.30.10",
                "1.31.0", "1.31.1", "1.31.2", "1.31.3", "1.31.4", "1.31.5", "1.31.6",
                "1.32.0", "1.32.1", "1.32.2"
            ]
        },
        {
            "short_name": "eks",
            "versions": [
                "1.25", "1.26", "1.27", "1.28", "1.29", "1.30", "1.31", "1.32"
            ]
        },
        {
            "short_name": "openshift",
            "versions": [
                "4.10.0-okd", "4.11.0-okd", "4.12.0-okd", "4.13.0-okd",
                "4.14.0-okd", "4.15.0-okd", "4.16.0-okd", "4.17.0-okd"
            ]
        }
    ]
};

// Test cases
function runTests() {
    console.log("Running tests...");

    // Test getLatestVersion
    testGetLatestVersion();
    
    // Test getLastMajorMinorVersions
    testGetLastMajorMinorVersions();

    // Test getLatestMinorVersions with different majorVersions configurations
    testGetLatestMinorVersionsWithSpecificMajors();
    testGetLatestMinorVersionsWithNullMajors();
    testGetLatestMinorVersionsWithEmptyMajors();

    console.log("All tests passed!");
}

function testGetLatestVersion() {
    console.log("\nTesting getLatestVersion...");

    // Test with k3s distribution
    const k3s = apiResponse["cluster-versions"].find(d => d.short_name === "k3s");
    const latestK3sVersion = getLatestVersion(k3s);
    console.log(`Latest k3s version: ${latestK3sVersion}`);
    assert.strictEqual(latestK3sVersion, "1.32.2", "Should find the latest k3s version");

    // Test with eks distribution
    const eks = apiResponse["cluster-versions"].find(d => d.short_name === "eks");
    const latestEksVersion = getLatestVersion(eks);
    console.log(`Latest eks version: ${latestEksVersion}`);
    assert.strictEqual(latestEksVersion, "1.32", "Should find the latest eks version");

    // Test with openshift distribution
    const openshift = apiResponse["cluster-versions"].find(d => d.short_name === "openshift");
    const latestOpenshiftVersion = getLatestVersion(openshift);
    console.log(`Latest openshift version: ${latestOpenshiftVersion}`);
    assert.strictEqual(latestOpenshiftVersion, "4.17.0-okd", "Should find the latest openshift version");

    console.log("getLatestVersion tests passed!");
}

function testGetLastMajorMinorVersions() {
    console.log("\nTesting getLastMajorMinorVersions...");

    const k3s = apiResponse["cluster-versions"].find(d => d.short_name === "k3s");
    
    // Test with different numbers of latest versions
    const top3Versions = getLastMajorMinorVersions(k3s, 3);
    console.log("Top 3 major.minor versions:", top3Versions);
    assert.strictEqual(top3Versions.length, 3, "Should return exactly 3 versions");
    assert.strictEqual(top3Versions[0], "1.32", "First version should be 1.32");
    assert.strictEqual(top3Versions[1], "1.31", "Second version should be 1.31");
    assert.strictEqual(top3Versions[2], "1.30", "Third version should be 1.30");
    
    // Test with 5 versions
    const top5Versions = getLastMajorMinorVersions(k3s, 5);
    console.log("Top 5 major.minor versions:", top5Versions);
    assert.strictEqual(top5Versions.length, 5, "Should return exactly 5 versions");
    assert.strictEqual(top5Versions[0], "1.32", "First version should be 1.32");
    assert.strictEqual(top5Versions[4], "1.28", "Fifth version should be 1.28");
    
    // Test with more versions than available
    const allVersions = getLastMajorMinorVersions(k3s, 20);
    console.log("All major.minor versions (requested 20):", allVersions);
    assert.strictEqual(allVersions.length, 9, "Should return all 9 available major.minor versions");
    assert.strictEqual(allVersions[0], "1.32", "First version should be 1.32");
    assert.strictEqual(allVersions[8], "1.24", "Last version should be 1.24");
    
    // Test with 0 versions
    const noVersions = getLastMajorMinorVersions(k3s, 0);
    console.log("No versions (requested 0):", noVersions);
    assert.strictEqual(noVersions.length, 0, "Should return an empty array");

    // Test with undefined versions
    const undefinedVersions = getLastMajorMinorVersions(k3s);
    console.log("Undefined versions (requested undefined):", undefinedVersions);
    assert.strictEqual(allVersions.length, 9, "Should return all 9 available major.minor versions");
    
    console.log("getLastMajorMinorVersions tests passed!");
}

function testGetLatestMinorVersionsWithSpecificMajors() {
    console.log("\nTesting getLatestMinorVersions with specific major versions...");

    const k3s = apiResponse["cluster-versions"].find(d => d.short_name === "k3s");

    // Test with specific major versions (1.30, 1.31, 1.32)
    const numOfLatestMajorMinorVersions = 3;
    const majorMinorVersionFilter = getLastMajorMinorVersions(k3s, numOfLatestMajorMinorVersions);
    const latestMinorVersions = getLatestMinorVersions(k3s, majorMinorVersionFilter);

    console.log("Latest minor versions for majors 1.30, 1.31, 1.32:");
    
    assert.strictEqual(Object.keys(latestMinorVersions).length, 3, "Should find the correct number of minor versions");
    assert.strictEqual(latestMinorVersions["1.30"], "1.30.10", "Should find latest 1.30.x version");
    assert.strictEqual(latestMinorVersions["1.31"], "1.31.6", "Should find latest 1.31.x version");
    assert.strictEqual(latestMinorVersions["1.32"], "1.32.2", "Should find latest 1.32.x version");

    console.log("getLatestMinorVersions with specific majors test passed!");
}

function testGetLatestMinorVersionsWithNullMajors() {
    console.log("\nTesting getLatestMinorVersions with null majorVersions...");

    const k3s = apiResponse["cluster-versions"].find(d => d.short_name === "k3s");

    // Test with null majorVersions
    const latestMinorVersions = getLatestMinorVersions(k3s, null);

    console.log("Number of minor versions found:", Object.keys(latestMinorVersions).length);

    // We should get all minor versions (1.24 through 1.32)
    assert.strictEqual(Object.keys(latestMinorVersions).length, 9, "Should find all 9 minor versions");
    assert.strictEqual(latestMinorVersions["1.24"], "1.24.17", "Should find latest 1.24.x version");
    assert.strictEqual(latestMinorVersions["1.25"], "1.25.16", "Should find latest 1.25.x version");
    assert.strictEqual(latestMinorVersions["1.26"], "1.26.15", "Should find latest 1.26.x version");
    assert.strictEqual(latestMinorVersions["1.27"], "1.27.16", "Should find latest 1.27.x version");
    assert.strictEqual(latestMinorVersions["1.28"], "1.28.15", "Should find latest 1.28.x version");
    assert.strictEqual(latestMinorVersions["1.29"], "1.29.14", "Should find latest 1.29.x version");
    assert.strictEqual(latestMinorVersions["1.30"], "1.30.10", "Should find latest 1.30.x version");
    assert.strictEqual(latestMinorVersions["1.31"], "1.31.6", "Should find latest 1.31.x version");
    assert.strictEqual(latestMinorVersions["1.32"], "1.32.2", "Should find latest 1.32.x version");

    console.log("getLatestMinorVersions with null majorVersions test passed!");
}

function testGetLatestMinorVersionsWithEmptyMajors() {
    console.log("\nTesting getLatestMinorVersions with empty majorVersions...");

    const k3s = apiResponse["cluster-versions"].find(d => d.short_name === "k3s");

    // Test with empty majorVersions
    const latestMinorVersions = getLatestMinorVersions(k3s, new Set());

    console.log("Number of minor versions found:", Object.keys(latestMinorVersions).length);

    // We should get all minor versions (1.24 through 1.32)
    assert.strictEqual(Object.keys(latestMinorVersions).length, 9, "Should find all 9 minor versions");
    assert.strictEqual(latestMinorVersions["1.24"], "1.24.17", "Should find latest 1.24.x version");
    assert.strictEqual(latestMinorVersions["1.32"], "1.32.2", "Should find latest 1.32.x version");

    console.log("getLatestMinorVersions with empty majorVersions test passed!");
}

// Run the tests
runTests(); 