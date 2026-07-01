import '../src/styles.css';
import 'filepond/dist/filepond.min.css';
import 'tabulator-tables/dist/css/tabulator_simple.min.css';
import Alpine from 'alpinejs';
import * as FilePond from 'filepond';
import { TabulatorFull as Tabulator } from 'tabulator-tables';
import { createApp } from './app/create-app.js';

if (typeof window !== 'undefined') {
    window.Alpine = Alpine;
    window.FilePond = FilePond;
    window.Tabulator = Tabulator;
    window.app = createApp;
    Alpine.start();
}

export { createApp };
