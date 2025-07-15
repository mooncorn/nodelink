import Docker from "dockerode";
import {
  DockerStartAction,
  DockerStopAction,
  DockerListAction,
  DockerStartResponse,
  DockerStopResponse,
  DockerListResponse,
} from "nodelink-shared";

export class DockerActions {
  private docker: Docker;

  constructor() {
    this.docker = new Docker();
  }

  async startContainer(
    action: DockerStartAction
  ): Promise<DockerStartResponse> {
    const {
      image,
      containerName,
      ports = [],
      environment = {},
      volumes = [],
    } = action;

    try {
      // Convert ports to proper format
      const formattedPorts = ports.map((port) => ({
        host: port.host,
        container: port.container,
        protocol: port.protocol || ("tcp" as const),
      }));

      // Prepare port bindings for dockerode
      const portBindings: { [key: string]: [{ HostPort: string }] } = {};
      const exposedPorts: { [key: string]: {} } = {};

      for (const port of ports) {
        const containerPort = `${port.container}/${port.protocol || "tcp"}`;
        portBindings[containerPort] = [{ HostPort: port.host.toString() }];
        exposedPorts[containerPort] = {};
      }

      // Prepare environment variables
      const envVars = Object.entries(environment).map(
        ([key, value]) => `${key}=${value}`
      );

      // Prepare volume bindings
      const binds = volumes.map((volume) => {
        const bind = `${volume.host}:${volume.container}`;
        return volume.mode ? `${bind}:${volume.mode}` : bind;
      });

      // Create container
      const createOptions: any = {
        Image: image,
        Env: envVars,
        ExposedPorts: exposedPorts,
        HostConfig: {
          PortBindings: portBindings,
          Binds: binds,
        },
      };

      if (containerName) {
        createOptions.name = containerName;
      }

      const container = await this.docker.createContainer(createOptions);

      // Start container
      await container.start();

      // Get container info
      const containerInfo = await container.inspect();

      return {
        containerId: containerInfo.Id,
        containerName: containerInfo.Name.replace("/", ""),
        status: containerInfo.State.Status,
        ports: formattedPorts,
      };
    } catch (error) {
      throw new Error(`Failed to start container: ${error}`);
    }
  }

  async stopContainer(action: DockerStopAction): Promise<DockerStopResponse> {
    const { containerId, force = false } = action;

    try {
      const container = this.docker.getContainer(containerId);

      if (force) {
        await container.kill();
      } else {
        await container.stop();
      }

      return {
        containerId,
        stopped: true,
      };
    } catch (error) {
      throw new Error(`Failed to stop container: ${error}`);
    }
  }

  async listContainers(action: DockerListAction): Promise<DockerListResponse> {
    const { all = false, filters = {} } = action;

    try {
      const options: any = { all };

      if (Object.keys(filters).length > 0) {
        options.filters = JSON.stringify(filters);
      }

      const containers = (await this.docker.listContainers(
        options
      )) as unknown as any[];

      const formattedContainers = containers.map((container: any) => ({
        id: container.Id,
        name: container.Names[0]?.replace("/", "") || "",
        image: container.Image,
        status: container.Status,
        ports:
          container.Ports?.map((port: any) => ({
            host: port.PublicPort || 0,
            container: port.PrivatePort,
            protocol: port.Type as "tcp" | "udp",
          })) || [],
        created: new Date(container.Created * 1000),
      }));

      return { containers: formattedContainers };
    } catch (error) {
      throw new Error(`Failed to list containers: ${error}`);
    }
  }

  async isDockerAvailable(): Promise<boolean> {
    try {
      await this.docker.ping();
      return true;
    } catch (error) {
      return false;
    }
  }
}
