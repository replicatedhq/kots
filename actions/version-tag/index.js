const core = require('@actions/core');

try {
    const githubRef = process.env.GITHUB_REF;
    if (githubRef.startsWith("refs/tags/")) {
        const tag = githubRef.slice(10);
        core.setOutput("GIT_TAG", tag);
    } else if (process.env.GITHUB_EVENT_NAME === "schedule") {
        var dateObj = new Date();
        var month = dateObj.getUTCMonth() + 1; // months are 0-based
        var day = dateObj.getUTCDate();
        var year = dateObj.getUTCFullYear();
        core.setOutput("GIT_TAG", `v${year}.${month}.${day}-nightly`);
    }
} catch (error) {
    core.setFailed(error.message);
}