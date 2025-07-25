import {EventSource} from 'eventsource';

import {argv} from 'process';
if (argv.length < 3) {
  console.error('Usage: node client.js <id>');
  process.exit(1);
}
const id = argv[2];

const createClient = (id: string) => {
  const es = new EventSource('http://localhost:8080/stream');

  es.onopen = () => {
    console.log(`[${id}] Connection opened`);
  }

  es.onmessage = (event) => {
    console.log(`[${id}] New message:`, event.data);
  };

  es.onerror = (error) => {
    console.error(`[${id}] Error occurred:`, error);
  };
}

createClient(id);

