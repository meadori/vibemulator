import os
import random
import math
from collections import deque
import numpy as np
import torch
import torch.nn as torch_nn
import torch.optim as optim
import torch.nn.functional as F
from vibemulator_env import VibemulatorEnv

# --- Hyperparameters ---
BATCH_SIZE = 32
GAMMA = 0.99
EPS_START = 1.0
EPS_END = 0.1
EPS_DECAY = 1000000
TARGET_UPDATE = 1000
MEMORY_SIZE = 50000
LR = 1e-4

# Determine device (MPS for Mac, CUDA for Nvidia, otherwise CPU)
device = torch.device(
    "cuda" if torch.cuda.is_available() else "mps" if torch.backends.mps.is_available() else "cpu"
)

# --- DQN Architecture ---
# This mimics the architecture from the Mnih et al. 2015 paper
class DQN(torch_nn.Module):
    def __init__(self, h, w, outputs):
        super(DQN, self).__init__()
        self.conv1 = torch_nn.Conv2d(3, 32, kernel_size=8, stride=4)
        self.conv2 = torch_nn.Conv2d(32, 64, kernel_size=4, stride=2)
        self.conv3 = torch_nn.Conv2d(64, 64, kernel_size=3, stride=1)

        # Number of Linear input connections depends on output of conv2d layers
        def conv2d_size_out(size, kernel_size=3, stride=1):
            return (size - (kernel_size - 1) - 1) // stride  + 1
        
        convw = conv2d_size_out(conv2d_size_out(conv2d_size_out(w, 8, 4), 4, 2), 3, 1)
        convh = conv2d_size_out(conv2d_size_out(conv2d_size_out(h, 8, 4), 4, 2), 3, 1)
        linear_input_size = convw * convh * 64
        
        self.fc1 = torch_nn.Linear(linear_input_size, 512)
        self.fc2 = torch_nn.Linear(512, outputs)

    def forward(self, x):
        x = F.relu(self.conv1(x))
        x = F.relu(self.conv2(x))
        x = F.relu(self.conv3(x))
        x = F.relu(self.fc1(x.view(x.size(0), -1)))
        return self.fc2(x)

# --- Experience Replay ---
class ReplayMemory(object):
    def __init__(self, capacity):
        self.memory = deque([], maxlen=capacity)

    def push(self, state, action, next_state, reward, done):
        self.memory.append((state, action, next_state, reward, done))

    def sample(self, batch_size):
        return random.sample(self.memory, batch_size)

    def __len__(self):
        return len(self.memory)

# --- Training Loop ---
def optimize_model(policy_net, target_net, memory, optimizer):
    if len(memory) < BATCH_SIZE:
        return
    
    transitions = memory.sample(BATCH_SIZE)
    # Transpose the batch
    batch_state, batch_action, batch_next_state, batch_reward, batch_done = zip(*transitions)

    # Convert to PyTorch tensors
    state_batch = torch.cat(batch_state).to(device)
    action_batch = torch.tensor(batch_action, device=device).unsqueeze(1)
    reward_batch = torch.tensor(batch_reward, device=device)
    done_batch = torch.tensor(batch_done, device=device, dtype=torch.float32)

    # Compute Q(s_t, a)
    state_action_values = policy_net(state_batch).gather(1, action_batch)

    # Compute V(s_{t+1}) for all next states.
    # Non-final states are ones where done == False
    non_final_mask = (1 - done_batch).bool()
    non_final_next_states = torch.cat([s for i, s in enumerate(batch_next_state) if not batch_done[i]]).to(device)
    
    next_state_values = torch.zeros(BATCH_SIZE, device=device)
    if non_final_next_states.shape[0] > 0:
        with torch.no_grad():
            next_state_values[non_final_mask] = target_net(non_final_next_states).max(1).values

    # Expected Q values
    expected_state_action_values = (next_state_values * GAMMA) + reward_batch

    # Compute Huber loss (Smooth L1)
    criterion = torch_nn.SmoothL1Loss()
    loss = criterion(state_action_values, expected_state_action_values.unsqueeze(1))

    # Optimize the model
    optimizer.zero_grad()
    loss.backward()
    torch_nn.utils.clip_grad_value_(policy_net.parameters(), 100)
    optimizer.step()

def main():
    print(f"Using device: {device}")
    env = VibemulatorEnv(state_file='vibemulator.sav')
    
    # Gym environments return [H, W, C], PyTorch CNNs expect [C, H, W]
    init_screen, _ = env.reset()
    init_screen = np.transpose(init_screen, (2, 0, 1))
    _, screen_height, screen_width = init_screen.shape

    # Get number of actions from gym action space
    n_actions = env.action_space.n

    policy_net = DQN(screen_height, screen_width, n_actions).to(device)
    target_net = DQN(screen_height, screen_width, n_actions).to(device)
    target_net.load_state_dict(policy_net.state_dict())

    optimizer = optim.AdamW(policy_net.parameters(), lr=LR, amsgrad=True)
    memory = ReplayMemory(MEMORY_SIZE)

    steps_done = 0
    num_episodes = 500

    print("Starting Training Loop...")
    for i_episode in range(num_episodes):
        state, info = env.reset()
        state = torch.tensor(np.transpose(state, (2, 0, 1)), dtype=torch.float32, device=device).unsqueeze(0) / 255.0
        
        total_reward = 0
        for t in range(10000): # max steps per episode
            # Select and perform an action
            sample = random.random()
            eps_threshold = EPS_END + (EPS_START - EPS_END) * math.exp(-1. * steps_done / EPS_DECAY)
            steps_done += 1
            
            if sample > eps_threshold:
                with torch.no_grad():
                    action = policy_net(state).max(1).indices.item()
            else:
                action = env.action_space.sample()

            next_state_np, reward, done, _, _ = env.step(action)
            total_reward += reward
            reward_tensor = torch.tensor([reward], device=device)

            if done:
                next_state = None
            else:
                next_state = torch.tensor(np.transpose(next_state_np, (2, 0, 1)), dtype=torch.float32, device=device).unsqueeze(0) / 255.0

            # Store the transition in memory
            memory.push(state, action, next_state, reward, done)

            # Move to the next state
            state = next_state

            # Perform one step of the optimization
            optimize_model(policy_net, target_net, memory, optimizer)

            # Soft update of the target network's weights
            if steps_done % TARGET_UPDATE == 0:
                target_net.load_state_dict(policy_net.state_dict())

            if done:
                break
                
        print(f"Episode {i_episode} completed. Total Reward: {total_reward:.2f}. Epsilon: {eps_threshold:.2f}")

    print('Training Complete')
    os.makedirs('models', exist_ok=True)
    torch.save(policy_net.state_dict(), 'models/dqn_nes.pth')

if __name__ == '__main__':
    main()
