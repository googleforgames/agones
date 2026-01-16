// Copyright 2026 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

const js = require('@eslint/js');
const globals = require('globals');

module.exports = [
	js.configs.recommended,
	{
		languageOptions: {
			ecmaVersion: 2018,
			globals: {
				...globals.node,
				...globals.es6,
				...globals.jasmine
			}
		},
		rules: {
			'indent': ['error', 'tab'],
			'lines-between-class-members': 'error',
			'quotes': ['error', 'single'],
			'semi': 'error',
			'space-before-blocks': 'error'
		}
	}
];
