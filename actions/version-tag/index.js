const core = require('@actions/core');

try {
    const githubRef = process.env.GITHUB_REF;
    if (githubRef.startsWith("refs/tags/")) {
        const tag = githubRef.slice(10);
        core.setOutput("GIT_TAG", tag);
        core.setOutput("IMAGE_TAG", tag);
    } else if (process.env.GITHUB_EVENT_NAME === "schedule") {
        core.setOutput("IMAGE_TAG", `nightly-${process.env.GITHUB_SHA.slice(0, 7)}`);
    }
} catch (error) {
    core.setFailed(error.message);
}