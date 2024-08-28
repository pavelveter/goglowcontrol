# **Go GlowControl**

### _A Powerful Go Utility for Your Yeelight Smart Bulbs_

![Go GlowControl](go_glowlogo.png)

Welcome to **Go GlowControl** — your ultimate Go utility to seamlessly control Yeelight smart bulbs over Wi-Fi. With a simple yet powerful command-line interface, GlowControl empowers you to light up your world exactly the way you want it.

## **Features**

- **Easy Control:** Manage single or multiple Yeelight bulbs with just a few keystrokes.
- **Customizable Lighting:** Set the perfect color and brightness to match any occasion.
- **Exciting Modes:** Engage disco mode for a party atmosphere or sunrise mode to start your day right.
- **Notifications:** Use your lights as a notification system with color-coded alerts.
- **Scenes:** Quickly set up lighting with predefined scenes.
- **Parallel Execution:** Control multiple bulbs simultaneously for efficient operation.

## **Getting Started**

### **Usage**

```
light <ip|@alias> <command>
```

**`<ip>`**: A single IP, multiple IPs, or a range of IP addresses to control specific bulbs.

**`<@alias>`**: An alias for a bulb or a group of bulbs (e.g., `@room`, `@kitchen`, `@all`).

**`<command>`**: The action you want to perform.

### **Commands**

- **`on`**: Turn on the light.
- **`off`**: Turn off the light.
- **`[color] <color>`**: Set the light to a specific color. The `color` key is optional.
- **`[t] <number>`**: Set the white light temperature (1700K to 6500K). The `t` key is optional.
- **`disco`**: Activate disco mode.
- **`sunrise`**: Activate sunrise mode.
- **`notify-<color>`**: Send a notification using a specified color.
- **`dim`**: Dim the light to 5% brightness.
- **`undim`**: Reset the light to 100% brightness.
- **`scene <name>`**: Execute a predefined scene.
- **`[brightness] <level>`**: Set the brightness level (1-100). The `brightness` key is optional.

### **Colors Available**

Choose from a wide range of colors, including:

Amber, Apricot, Aquamarine, Azure, Beige, Blue, Burgundy, Byzantium, Carmine, Cerise, Cerulean, Chartreuse, Coral, Crimson, Cyan, Dandelion, Denim, Desert, Emerald, Erin, Flamingo, Forest, Green, Grey, Indigo, Ivory, Jade, Lavender, Lemon, Lilac, Lime, Magenta, Mauve, Navy, Olive, Orange, Orchid, Peach, Periwinkle, Plum, Purple, Quartz, Red, Rose, Ruby, Saffron, Salmon, Sapphire, Scarlet, Sepia, Sienna, Silver, Sky, Tan, Teal, Turquoise, Ultramarine, Violet, Xanadu, Yellow, Zaffre

### **Aliases Available**

Control groups of bulbs with ease:

- `@kitchen`, `@bathroom`, `@monitor`, `@stand`, `@tv`, `@room`, `@all`

### **Scenes Available**

Quickly set up lighting with predefined scenes:

- `red`, `evening`, `lr`, `work`, `movie`

### **Examples**

1. **Turn on a single bulb:**
   ```
   light 192.168.88.1 on
   ```
2. **Set three bulbs to red:**
   ```
   light 192.168.88.1-2 192.168.88.4 color red
   ```
3. **Adjust brightness of two bulbs:**
   ```
   light 192.168.88.1 192.168.88.3 50
   ```
4. **Set white temperature to 4100K:**
   ```
   light 192.168.88.2 t 4100
   ```
5. **Notify via room bulbs with blue color:**
   ```
   light @room notify-blue
   ```
6. **Execute the evening scene:**
   ```
   light scene evening
   ```

## **Installation**

Clone the repository and build the Go program:

```bash
git clone https://github.com/pavelveter/goglowcontrol.git
cd goglowcontrol
go build -o light
```
## **Configuration Files**

Go GlowControl uses several text files for configuration:

### **colors.txt**
Contains a list of available colors and their hexadecimal values. Each line has the format:
```
color_name;hex_value
```

### **bulbs.txt**
Contains a list of available bulbs. Each line has the format:
```
bulb_alias: ip_address1 ip_address2 ...
```
### **scenes.txt**
Contains a list of predefined scenes. Each line has the format:
```
scene_name: bulb_alias1 action1, bulb_alias2 action2, ..., bulb_aliasN actionN
```

These files should be located in the same directory as the GlowControl executable or in the current working directory.

## **Contributing**

We welcome contributions! Feel free to submit issues, fork the repo, and send pull requests.

## **License**

Go GlowControl is licensed under the MIT License.

---

Illuminate your life with the power of Go and Yeelight — experience Go GlowControl today!

p.s. Based on https://github.com/shyamvalsan/YeelightController