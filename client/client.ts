import {EventSource} from 'eventsource';

const serverURL = "http://localhost:8080"

const executeShell = async (agentId: string, cmd: string) => {
  try {
    const response = await fetch(`${serverURL}/agents/${agentId}/shell`, {
      method: "POST",
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({cmd})
    })

    const data = await response.json()
    const ref = data.ref
    console.log("Ref: ", ref)

    // listen to result using ref
    const es = new EventSource(`${serverURL}/stream?ref=${ref}`);

    es.onopen = () => {
      console.log(`[${ref}] Connection opened`);
    }

    es.onmessage = (event) => {
      console.log(`[${ref}] New message:`, event.data);
    };

    es.addEventListener("response", (event) => {
      console.log(`[${ref}] New response:`, event.data);
    });

    es.onerror = (error) => {
      console.error(`[${ref}] Error occurred:`, error);
    };

  } catch (error) {
    console.error('Error executing shell command:', error);
  }
}

executeShell("agent1", "ls")