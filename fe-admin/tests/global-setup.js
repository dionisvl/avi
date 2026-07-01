// @ts-check
const { loginAndSaveState, loginAndSaveStateAs, FIXTURE_USER_ROLE_USER_ONLY } = require('./helpers');

module.exports = async function globalSetup() {
    await loginAndSaveState();
    await loginAndSaveStateAs(FIXTURE_USER_ROLE_USER_ONLY, '.auth.fixture-user-only.json');
};
