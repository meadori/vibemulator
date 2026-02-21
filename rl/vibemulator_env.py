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

        # A truly generic action space: 8 independent buttons (2^8 = 256 possible combinations)
        # Bit 0: A, Bit 1: B, Bit 2: Select, Bit 3: Start, Bit 4: Up, Bit 5: Down, Bit 6: Left, Bit 7: Right
        self.action_space = spaces.Discrete(256)

        # Observation space: 256x240 RGB image
        self.observation_space = spaces.Box(low=0, high=255,
                                            shape=(240, 256, 3), dtype=np.uint8)

        # Track the last observation for generic pixel-delta rewards
        self._last_obs = np.zeros((240, 256, 3), dtype=np.uint8)

        # gRPC Setup
        self.channel = grpc.insecure_channel(self.host)
        self.stub = controller_pb2_grpc.ControllerServiceStub(self.channel)
        
        # We need a stream to send inputs. We use a generator that yields the current action.
        # FIX: The agent must "hold" the button during the environment step, not immediately revert to NOOP.
        self._current_action = controller_pb2.InputState(player_index=1)
        
        def input_generator():
            while True:
                yield self._current_action
                time.sleep(1/60.0) # Sync roughly with 60fps

        self.input_stream = self.stub.StreamInput(input_generator())

    def _action_to_proto(self, action):
        state = controller_pb2.InputState(player_index=1)
        # Action Mapping: Bitwise decode of the 0-255 integer
        state.a      = bool(action & 0b00000001)
        state.b      = bool(action & 0b00000010)
        state.select = bool(action & 0b00000100)
        state.start  = bool(action & 0b00001000)
        state.up     = bool(action & 0b00010000)
        state.down   = bool(action & 0b00100000)
        state.left   = bool(action & 0b01000000)
        state.right  = bool(action & 0b10000000)
        return state

    def step(self, action):
        # 1. Update the held action
        self._current_action = self._action_to_proto(action)

        # 2. Wait a tick (simplified sync, in a real env we'd wait for the emulator to signal frame ready)
        time.sleep(1/60.0)

        # 3. Get Observation
        obs = self._get_obs()

        # 4. Get generic Reward (based on screen changes) and Done state
        reward, done = self._get_reward_and_done(obs)
        self._last_obs = obs

        info = {}
        truncated = False

        if self.render_mode == "human":
            self.render()

        return obs, reward, done, truncated, info

    def reset(self, seed=None, options=None):
        super().reset(seed=seed)
        
        # Reset the physical NES console (back to the title screen)
        if not self.state_file:
            try:
                self.stub.ResetSystem(controller_pb2.Empty())
                # Sleep a bit to let the console reboot
                time.sleep(1/10.0) 
            except grpc.RpcError as e:
                print(f"Failed to reset system: {e}")
        else:
            # Load a specific emulator save state
            try:
                self.stub.LoadState(controller_pb2.StateRequest(filename=self.state_file))
                # Give emulator a frame to process the load
                time.sleep(1/60.0)
            except grpc.RpcError as e:
                print(f"Failed to load state {self.state_file}: {e}")

        obs = self._get_obs()
        self._last_obs = obs
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

    def _get_reward_and_done(self, obs):
        # GENERIC REWARD: "Pixel Delta" / "Curiosity"
        # Since we don't know the game's specific RAM map (e.g. for score/lives), 
        # we reward the agent for causing the screen to change.
        # This naturally incentivizes pressing Start and exploring new areas.
        
        diff = np.abs(obs.astype(np.int32) - self._last_obs.astype(np.int32))
        delta = np.mean(diff)
        
        # Scale the reward slightly
        reward = float(delta) * 0.05
        
        # Since we are generic, we never strictly 'die', the episode just times out.
        # (A user must supply a custom done condition for specific games)
        done = False
                
        return reward, done

