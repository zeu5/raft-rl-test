import numpy as np
import matplotlib.pyplot as plt
import os
import math


class Position:
    def __init__(self, x, y, z, grid):
        self.x = x
        self.y = y
        self.z = z
        self.grid = grid

    def __eq__(self, other):
        return self.x == other.x and self.y == other.y and self.z == other.z and self.grid == other.grid

    def __str__(self):
        return f"({self.x}, {self.y}, {self.z}, {self.grid})"

    def __repr__(self):
        return f"({self.x}, {self.y}, {self.z}, {self.grid})"

    def copy(self):
        return Position(self.x, self.y, self.z, self.grid)

    def __hash__(self):
        return hash((self.x, self.y, self.z, self.grid))

class Grid:
    def __init__(self, grids, height, width, depth, doors = {}):
        self.grids = grids
        self.height = height
        self.width = width
        self.depth = depth
        # (x, y, z, grid)
        self._cur_pos = Position(0, 0, 0, 0)
        self.doors = doors

    def _is_door(self, pos):
        if pos in self.doors:
            return True

    def move(self, direction):
        pos = self._cur_pos
        if direction == "up":
            if pos.z < self.depth-1:
                pos.z += 1
        elif direction == "down":
            if pos.z > 0:
                pos.z -= 1
        elif direction == "left":
            if pos.y > 0:
                pos.y -= 1
        elif direction == "right":
            if pos.y < self.width-1:
                pos.y += 1
        elif direction == "forward":
            if pos.x < self.height-1:
                pos.x += 1
        elif direction == "backward":
            if pos.x > 0:
                pos.x -= 1
        elif direction == "into" and self._is_door(self._cur_pos):
            pos = self.doors[pos]

        self._cur_pos = pos
        return self._cur_pos.copy()
    
    def directions(self, pos):
        directions = []
        if pos.z < self.depth-1:
            directions.append("up")
        if pos.z > 0:
            directions.append("down")
        if pos.y > 0:
            directions.append("left")
        if pos.y < self.width-1:
            directions.append("right")
        if pos.x < self.height-1:
            directions.append("forward")
        if pos.x > 0:
            directions.append("backward")
        if self._is_door(pos):
            directions.append("into")
        return directions
    
    def all_directions(self):
        return ["up", "down", "left", "right", "forward", "backward", "into"]
        
    def reset(self):
        self._cur_pos = Position(0, 0, 0, 0)
        return self._cur_pos
    

class Agent:
    def __init__(self, name, env, policy) -> None:
        self.env = env
        self.policy = policy
        self.name = name
        self.traces = []
    
    def run(self,episodes, horizon):
        for episode in range(episodes):
            pos = self.env.reset()
            trace = []
            self.policy.reset_iter()
            for step in range(horizon):
                actions = self.env.directions(pos)
                action = self.policy.pick(pos, actions)
                next_pos = self.env.move(action)
                self.policy.update_state(pos, action, next_pos)
                trace.append((pos.copy(), action, next_pos.copy()))
                print(f"\r{self.name} - Episode: {episode+1}/{episodes}, Step: {step+1}/{horizon}",end="")
                pos = next_pos
            self.policy.update(trace)
            self.traces.append(trace)
        print("")
        return self.traces
    

class RandomPolicy:
    def __init__(self) -> None:
        pass

    def reset_iter(self):
        pass
    
    def pick(self, pos, actions):
        return np.random.choice(actions)

    def update(self, trace):
        pass

    def update_state(self, pos, action, next_pos):
        pass

class NegRewardPolicy:
    def __init__(self, alpha, gamma, actions) -> None:
        self.alpha = alpha
        self.gamma = gamma
        self.actions = actions
        self.q = {}
        self.visits = {}

    def reset_iter(self):
        pass

    def update_state(self, pos, action, next_pos):
        pass

    def pick(self, pos, actions):
        q_values = [self.q.get((pos, a), 0) for a in actions]
        # normalize q_values
        norm_q_values = q_values + np.min(q_values)
        exp_values = np.exp(norm_q_values)
        sum_exp_values = np.sum(exp_values)
        if sum_exp_values == 0:
            return np.random.choice(actions)
        
        probs = exp_values / sum_exp_values
        return np.random.choice(actions, p=probs)
    
    def update(self, trace):
        for (pos, action, next_pos) in trace:
            q_value = self.q.get((pos, action), 0)
            t = self.visits.get((next_pos, action), 0) +1
            self.visits[(next_pos, action)] = t
            next_q_value = max([self.q.get((next_pos, a), 0) for a in self.actions])
            self.q[(pos, action)] = (1-self.alpha)* q_value + self.alpha * ( -t + self.gamma * next_q_value)


class BonusMaxPolicy:
    def __init__(self, alpha, gamma, epsilon, actions) -> None:
        self.alpha = alpha
        self.gamma = gamma
        self.actions = actions
        self.epsilon = epsilon
        self.visits = {}
        self.q = {}
    
    def reset_iter(self):
        pass

    def update_state(self, pos, action, next_pos):
        pass

    def pick(self, pos, actions):
        if np.random.rand() < self.epsilon:
            return np.random.choice(actions)
        q_values = [self.q.get((pos, a), 1) for a in actions]
        return actions[np.argmax(q_values)]
    
    
    def update(self, trace):
        for (i,(pos, action, next_pos)) in enumerate(trace):
            q_value = self.q.get((pos, action), 1)
            t = self.visits.get((next_pos, action), 0) + 1
            self.visits[(next_pos, action)] = t
            next_q_value = max([self.q.get((next_pos, a), 1) for a in self.actions])
            if i == len(trace)-1:
                next_q_value = max([self.q.get((next_pos, a), 0) for a in self.actions])

            self.q[(pos, action)] = (1-self.alpha)* q_value + self.alpha * max(1/t, self.gamma * next_q_value)

class Predicate:
    def __init__(self, name, func) -> None:
        self.name = name
        self.func = func

    def __call__(self, pos) -> bool:
        return self.func(pos)

class PredHRLPolicy:
    def __init__(self, alpha, gamma, epsilon, actions, predicates, one_time = False) -> None:
        predicates = [Predicate("init", lambda pos: True)] + predicates
        self.alpha = alpha
        self.gamma = gamma
        self.epsilon = epsilon
        self.actions = actions
        self.predicates = predicates
        self.one_time = one_time

        self.q_tables = {}
        for pred in predicates:
            self.q_tables[pred.name] = {}

        self.visit_tables = {}
        for pred in predicates:
            self.visit_tables[pred.name] = {}

        self._cur_pred_name = predicates[0].name
        self._cur_pred_index = 0
        self._trace_segments = {pred.name: [] for pred in predicates}
        self._target_reached = False

    def reset_iter(self):
        self._cur_pred_name = self.predicates[0].name
        self._cur_pred_index = 0
        self._trace_segments = {pred.name: [] for pred in self.predicates}
        self._target_reached = False
    
    def pick(self, pos, actions):
        if np.random.rand() < self.epsilon:
            return np.random.choice(actions)
        q_values = [self.q_tables[self._cur_pred_name].get((pos, a), 1) for a in actions]
        return actions[np.argmax(q_values)]
    
    def update_state(self, pos, action, next_pos):
        reward = False
        out_of_space = False
        cur_pred = self._cur_pred_name
        if not self._target_reached:
            next_pred_index = 0
            for index, pred in list(enumerate(self.predicates))[::-1]:
                if pred(next_pos):
                    next_pred_index = index
                    break
            if next_pred_index != self._cur_pred_index:
                out_of_space = True
            if next_pred_index > self._cur_pred_index:
                reward = True
            self._cur_pred_index = next_pred_index
            self._cur_pred_name = self.predicates[self._cur_pred_index].name

            if self.one_time and next_pred_index == len(self.predicates)-1:
                self._target_reached = True
        
        self._trace_segments[cur_pred].append((pos, action, next_pos, reward, out_of_space))

    def update(self, trace):
        for pred in self.predicates:
            trace_seg_len = len(self._trace_segments[pred.name])
            for (i, (pos, action, next_pos, reward, out_of_space)) in enumerate(self._trace_segments[pred.name]):
                q_value = self.q_tables[pred.name].get((pos, action), 1)
                t = self.visit_tables[pred.name].get((next_pos, action), 0) + 1
                self.visit_tables[pred.name][(next_pos, action)] = t
                next_q_value = max([self.q_tables[pred.name].get((next_pos, a), 1) for a in self.actions])
                if out_of_space or i == trace_seg_len-1:
                    next_q_value = max([self.q_tables[pred.name].get((next_pos, a), 0) for a in self.actions])
                
                r = 2+(1/t) if reward else 1/t
                self.q_tables[pred.name][(pos, action)] = (1-self.alpha)* q_value + self.alpha * max(r, self.gamma * next_q_value)

class Experiment:
    def __init__(self, name, env, policy) -> None:
        self.env = env
        self.name = name
        self.agent = Agent(name, env, policy)
        self.traces = []
    
    def run(self, episodes, horizon):
        self.traces = self.agent.run(episodes, horizon)
        return self.traces
    
    def plot(self):
        # plot a heatmap for each grid viewed from the top. The color indicates the combined visits for all positions in that depth.
        visits = np.zeros((self.env.height, self.env.width, self.env.grids))
        for trace in self.traces:
            for (pos, action, next_pos) in trace:
                visits[pos.x, pos.y, pos.grid] += 1

        # use subplots to plot each grid with 3 grids in a row
        rows = math.ceil(self.env.grids/3)
        fig, axs = plt.subplots(rows, 3)
        for i in range(self.env.grids):
            x,y = int(i//3), i%3
            axs[x,y].imshow(visits[:, :, i], cmap='hot', interpolation='nearest')
            axs[x,y].set_title(f"Grid {i}")
        
        plt.savefig(f"results/{self.name}.png")

    def plot_grid(self, grid):
        visits = np.zeros((self.env.height, self.env.width, self.env.depth))
        for trace in self.traces:
            for (pos, action, next_pos) in trace:
                if pos.grid == grid:
                    visits[pos.x, pos.y, pos.z] += 1
        
        rows = math.ceil(self.env.depth/3)
        fig, axs = plt.subplots(rows, 3)
        for i in range(self.env.depth):
            x,y = int(i//3), i%3
            if x > 1:
                axs[x,y].imshow(visits[:, :, i], cmap='hot', interpolation='nearest')
                axs[x,y].set_title(f"Depth {i}")
            else:
                axs[y].imshow(visits[:, :, i], cmap='hot', interpolation='nearest')
                axs[y].set_title(f"Depth {i}")

        plt.savefig(f"results/{self.name}_grid_{grid}.png")

def main():
    grids = 6
    height = 10
    width = 10
    depth = 3
    doors = {
        Position(5, 5, 2, 0): Position(0, 0, 0, 1), 
        Position(5, 5, 2, 1): Position(0, 0, 0, 2),
        Position(5, 5, 2, 2): Position(0, 0, 0, 3)
    }
    env = Grid(grids, height, width, depth, doors)

    # create a results folder, clear if it exists
    if not os.path.exists("results"):
        os.makedirs("results")
    else:
        for file in os.listdir("results"):
            os.remove(f"results/{file}")
    
    
    policy = NegRewardPolicy(0.1, 0.99, env.all_directions())
    exp = Experiment("NegReward", env, policy)
    exp.run(10000, 100)
    exp.plot()

    policy = BonusMaxPolicy(0.1, 0.99, 0.025, env.all_directions())
    exp = Experiment("BonusMax", env, policy)
    exp.run(10000, 100)
    exp.plot()
    exp.plot_grid(3)

    policy = RandomPolicy()
    exp = Experiment("Random", env, policy)
    exp.run(10000, 100)
    exp.plot()

    predicates = [
        Predicate("Grid1", lambda pos: pos.grid == 1),
        Predicate("Grid2", lambda pos: pos.grid == 2),
        Predicate("Grid3", lambda pos: pos.grid == 3)
    ]
    policy = PredHRLPolicy(0.1, 0.99, 0.025, env.all_directions(), predicates, one_time=True)
    exp = Experiment("PredHRL", env, policy)
    exp.run(10000, 100)
    exp.plot()
    exp.plot_grid(3)

if __name__ == "__main__":
    main()