import time
import numpy as cv2
import gymnasium as gym
from gymnasium import spaces
import numpy as np
import grpc

import sys
import os
# Adjust path to import the generated protobuf modules
sys.path.append(os.path.dirname(os.path.abspath(__file__)))

from api import controller_pb2
from api import controller_pb2_grpc


class VibemulatorEnv(gym.Env):
    """
    Custom Environment that follows gym interface, connecting to the Vibemulator via gRPC.
    """
    metadata = {'render_modes': ['human', 'rgb_array'], 'render_fps': 60}

    def __init__(self, host='localhost:50051', render_mode=None, state_file=None):
        super(VibemulatorEnv, self).__init__()
        self.render_mode = render_mode
        self.host = host
        self.state_file = state_file

        # We define a discrete action space for typical NES button combinations
        # 0: NOOP, 1: Right, 2: Right+A, 3: Right+B, 4: Right+A+B, 5: Left, 6: A, 7: B
        self.action_space = spaces.Discrete(8)

        # Observation space: 256x240 RGB image
        self.observation_space = spaces.Box(low=0, high=255,
                                            shape=(240, 256, 3), dtype=np.uint8)

        # gRPC Setup
        self.channel = grpc.insecure_channel(self.host)
        self.stub = controller_pb2_grpc.ControllerServiceStub(self.channel)
        
        # We need a stream to send inputs. We use a generator that yields the current action.
        self._action_queue = []
        def input_generator():
            while True:
                if self._action_queue:
                    yield self._action_queue.pop(0)
                else:
                    # Default NOOP
                    yield controller_pb2.InputState(player_index=1)
                time.sleep(1/60.0) # Sync roughly with 60fps

        self.input_stream = self.stub.StreamInput(input_generator())

    def _action_to_proto(self, action):
        state = controller_pb2.InputState(player_index=1)
        # Action Mapping
        if action == 1: state.right = True
        elif action == 2: state.right = True; state.a = True
        elif action == 3: state.right = True; state.b = True
        elif action == 4: state.right = True; state.a = True; state.b = True
        elif action == 5: state.left = True
        elif action == 6: state.a = True
        elif action == 7: state.b = True
        return state

    def step(self, action):
        # 1. Send the action
        proto_action = self._action_to_proto(action)
        self._action_queue.append(proto_action)

        # 2. Wait a tick (simplified sync, in a real env we'd wait for the emulator to signal frame ready)
        time.sleep(1/60.0)

        # 3. Get Observation
        obs = self._get_obs()

        # 4. Get Reward and Done state from RAM
        reward, done = self._get_reward_and_done()

        info = {}
        truncated = False

        if self.render_mode == "human":
            self.render()

        return obs, reward, done, truncated, info

    def reset(self, seed=None, options=None):
        super().reset(seed=seed)
        
        # Load a specific emulator save state (e.g. bypassing the title screen)
        if self.state_file:
            try:
                self.stub.LoadState(controller_pb2.StateRequest(filename=self.state_file))
                # Give emulator a frame to process the load
                time.sleep(1/60.0)
            except grpc.RpcError as e:
                print(f"Failed to load state {self.state_file}: {e}")

        obs = self._get_obs()
        return obs, {}

    def render(self):
        if self.render_mode == "rgb_array":
            return self._get_obs()
        elif self.render_mode == "human":
            # The emulator has its own window, but we could render via CV2 if we wanted.
            pass

    def close(self):
        self.channel.close()

    def _get_obs(self):
        try:
            response = self.stub.GetFrame(controller_pb2.Empty())
            # Convert raw RGBA bytes (256x240 * 4) to RGB Numpy array
            raw_bytes = np.frombuffer(response.pixels, dtype=np.uint8)
            # Ensure shape is correct (240 height, 256 width, 4 channels)
            if len(raw_bytes) == 256 * 240 * 4:
                rgba = raw_bytes.reshape((240, 256, 4))
                # Drop Alpha channel for standard RGB observation
                rgb = rgba[:, :, :3]
                return rgb
            else:
                return np.zeros((240, 256, 3), dtype=np.uint8)
        except grpc.RpcError as e:
            print(f"gRPC Error getting frame: {e}")
            return np.zeros((240, 256, 3), dtype=np.uint8)

    def _get_reward_and_done(self):
        # Example for Super Mario Bros:
        # X Position: 0x0086 (page) and 0x03AD (x within page)
        # Player state: 0x000E (0x0B means dying)
        try:
            page_resp = self.stub.ReadMemory(controller_pb2.MemoryRequest(address=0x0086))
            x_resp = self.stub.ReadMemory(controller_pb2.MemoryRequest(address=0x03AD))
            state_resp = self.stub.ReadMemory(controller_pb2.MemoryRequest(address=0x000E))
            
            x_pos = (page_resp.data * 256) + x_resp.data
            
            # Simple reward: moving right
            # (In a real implementation, track previous x_pos to compute delta)
            reward = 0.1 # Small reward for surviving
            
            done = False
            if state_resp.data == 0x0B: # Mario is dying
                reward = -10.0
                done = True
                
            return float(reward), done
        except grpc.RpcError:
            return 0.0, False

