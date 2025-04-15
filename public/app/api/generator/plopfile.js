const path = require('path');

const { projectPath: createProjectPath, formatOperationIds, validateGroup, validateVersion } = require('./helpers');

module.exports = function (plop) {
  // Grafana root path
  const basePath = path.resolve(__dirname, '../../../..');
  // Create project path helper with the base path
  const projectPath = createProjectPath(basePath);

  // List of created or modified files
  const getFilesToFormat = (groupName) => [
    `public/app/api/clients/${groupName}/baseAPI.ts`,
    `public/app/api/clients/${groupName}/index.ts`,
    `scripts/generate-rtk-apis.ts`,
    `public/app/core/reducers/root.ts`,
    `public/app/store/configureStore.ts`,
  ];

  plop.setActionType('runGenerateApis', function () {
    const { execSync } = require('child_process');
    try {
      console.log('⏳ Running yarn generate-apis to generate endpoints...');
      execSync('yarn generate-apis', { stdio: 'inherit', cwd: basePath });
      return '✅ API endpoints generated successfully!';
    } catch (error) {
      console.error('❌ Failed to generate API endpoints:', error.message);
      return '❌ Failed to generate API endpoints. See error above.';
    }
  });

  // Custom action function for formatting files with prettier and eslint
  plop.setActionType('formatFiles', function (_, config) {
    const { execSync } = require('child_process');
    const filesToFormat = config.files.map((file) => projectPath(file));

    try {
      const filesList = filesToFormat.map((file) => `"${file}"`).join(' ');

      console.log('🧹 Running ESLint on generated/modified files...');
      try {
        execSync(`yarn eslint --fix ${filesList}`, { cwd: basePath });
      } catch (error) {
        console.warn(`⚠️ Warning: ESLint encountered issues: ${error.message}`);
      }

      console.log('🧹 Running Prettier on generated/modified files...');
      try {
        execSync(`yarn prettier --write ${filesList}`, { cwd: basePath });
      } catch (error) {
        console.warn(`⚠️ Warning: Prettier encountered issues: ${error.message}`);
      }

      return '✅ Files linted and formatted successfully!';
    } catch (error) {
      console.error('⚠️ Warning: Formatting operations failed:', error.message);
      return '⚠️ Warning: Formatting operations failed.';
    }
  });

  // Register the helper used in templates
  plop.setHelper('formatOperationIds', formatOperationIds(plop));

  // Helper function to generate actions for creating RTK API client
  const generateRtkApiActions = (data) => {
    const { reducerPath, groupName } = data;

    return [
      // Create baseAPI.ts
      {
        type: 'add',
        path: projectPath(`public/app/api/clients/${groupName}/baseAPI.ts`),
        templateFile: './templates/baseAPI.ts.hbs',
      },

      {
        type: 'modify',
        path: projectPath('scripts/generate-rtk-apis.ts'),
        pattern: '// PLOP_INJECT_API_CLIENT',
        templateFile: './templates/config-entry.hbs',
      },

      // Create index.ts
      {
        type: 'add',
        path: projectPath(`public/app/api/clients/${groupName}/index.ts`),
        templateFile: './templates/index.ts.hbs',
      },

      // Update reducers and middleware
      {
        type: 'modify',
        path: projectPath('public/app/core/reducers/root.ts'),
        pattern: '// PLOP_INJECT_IMPORT',
        template: `import { ${reducerPath} } from '../../api/clients/${groupName}';\n// PLOP_INJECT_IMPORT`,
      },
      {
        type: 'modify',
        path: projectPath('public/app/core/reducers/root.ts'),
        pattern: '// PLOP_INJECT_REDUCER',
        template: `[${reducerPath}.reducerPath]: ${reducerPath}.reducer,\n  // PLOP_INJECT_REDUCER`,
      },
      {
        type: 'modify',
        path: projectPath('public/app/store/configureStore.ts'),
        pattern: '// PLOP_INJECT_IMPORT',
        template: `import { ${reducerPath} } from '../api/clients/${groupName}';\n// PLOP_INJECT_IMPORT`,
      },
      {
        type: 'modify',
        path: projectPath('public/app/store/configureStore.ts'),
        pattern: '// PLOP_INJECT_MIDDLEWARE',
        template: `${reducerPath}.middleware,\n        // PLOP_INJECT_MIDDLEWARE`,
      },

      // Format the generated files
      {
        type: 'formatFiles',
        files: getFilesToFormat(groupName),
      },

      // Run yarn generate-apis to generate endpoints
      {
        type: 'runGenerateApis',
      },
    ];
  };

  plop.setGenerator('rtk-api-client', {
    description: 'Generate RTK Query API client for a Grafana API group',
    prompts: [
      {
        type: 'input',
        name: 'groupName',
        message: 'API group name (e.g. dashboard):',
        validate: (input) => (input && input.trim() ? true : 'Group name is required'),
      },
      {
        type: 'input',
        name: 'group',
        message: 'API group (e.g. dashboard.grafana.app):',
        default: (answers) => `${answers.groupName}.grafana.app`,
        validate: validateGroup,
      },
      {
        type: 'input',
        name: 'version',
        message: 'API version (e.g. v0alpha1):',
        default: 'v0alpha1',
        validate: validateVersion,
      },
      {
        type: 'input',
        name: 'reducerPath',
        message: 'Reducer path (e.g. dashboardAPI):',
        default: (answers) => `${answers.groupName}API`,
        validate: (input) =>
          input && input.endsWith('API') ? true : 'Reducer path should end with "API" (e.g. dashboardAPI)',
      },
      {
        type: 'input',
        name: 'operationIds',
        message: 'Operation IDs to include (comma-separated, optional):',
        validate: () => true,
      },
    ],
    actions: function (data) {
      // Format data for templates
      data.operationIdArray = data.operationIds
        ? data.operationIds
            .split(',')
            .map((id) => id.trim())
            .filter(Boolean)
        : [];

      // Generate actions
      return generateRtkApiActions(data);
    },
  });
};
