package main

import (
  "flag"
  "log"
  "strings"
  "regexp"
  "os"
  "os/signal"
  "syscall"

  "github.com/bwmarrin/discordgo"
)

var (
  token string
  ownerID string = "0"
  activeChannel string = "roles"
  verbose bool = false
  emoji string = "🍆"
)

func init() {
  flag.StringVar(&token, "t", "", "Bot `token` (required)")
  flag.StringVar(&ownerID, "o", "", "Owner user `id`")
  flag.StringVar(&activeChannel, "c", "roles", "Channel `name` to use")
  flag.StringVar(&emoji, "e", "🍆", "Emoji to use as reaction button")
  flag.BoolVar(&verbose, "v", false, "Verbose logging")
  flag.Parse()
  if token == "" {
    flag.Usage()
    os.Exit(1)
  }
}

func debug(v ...interface{}) {
  if verbose {
    fa := "Debug: "
    v = append([]interface{}{fa}, v...)
    log.Print(v...)
  }
}

func main()  {
  discord, err := discordgo.New("Bot " + token)
  if err != nil {
    log.Fatal("error creating Discord session,", err)
    return
  }
  discord.AddHandler(messageCreate)
  discord.AddHandler(messageReactionAdd)
  discord.AddHandler(messageReactionRemove)
  // discord.AddHandler(ready)

  err = discord.Open()
	if err != nil {
		log.Fatal("error opening connection,", err)
		return
	}
  guilds, err := discord.UserGuilds(100, "", "")
  log.Print("Running on servers:")
  if len(guilds) == 0 {
    log.Print("\t(none)")
  }
  for index := range guilds {
    guild := guilds[index]
    log.Print("\t", guild.Name, " (", guild.ID, ")")
  }
  log.Print("channel name: ", activeChannel)
  log.Print("Join URL:")
  log.Print("https://discordapp.com/api/oauth2/authorize?scope=bot&permissions=268446720&client_id=", discord.State.User.ID)

  user, err := discord.User("@me")
  if err != nil {
    log.Print("Bot running. CTRL-C to exit.")
  } else {
    log.Print("Bot running as ", user.Username, "#", user.Discriminator, ". CTRL-C to exit.")
  }

  sc := make(chan os.Signal, 1)
  signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
  <-sc

  discord.Close()
}

func getIDsfromMsg(s *discordgo.Session, MessageID *string, ChannelID *string) (bool, string, string)  {
  channel, _ := s.Channel(*ChannelID)
  if channel.Name != "roles" {
    return false, "", ""
  }
  message, _ := s.ChannelMessage(*ChannelID, *MessageID)

  getRole := regexp.MustCompile(`<@&([0-9]+)>`)
  roleID := getRole.FindStringSubmatch(message.Content)[1]

  return true, channel.GuildID, roleID
}

func messageReactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd)  {
  shouldRun, GuildID, roleID := getIDsfromMsg(s, &r.MessageID, &r.ChannelID)
  if !shouldRun || r.UserID == s.State.User.ID {
    return
  }
  debug("Giving ", roleID, " to ", r.UserID)
  err := s.GuildMemberRoleAdd(GuildID, r.UserID, roleID)
  if err != nil {
    log.Print(err)
    debug("try moving the bot's role up the role list")
  }
}

func messageReactionRemove(s *discordgo.Session, r *discordgo.MessageReactionRemove)  {
  shouldRun, GuildID, roleID := getIDsfromMsg(s, &r.MessageID, &r.ChannelID)
  if !shouldRun || r.UserID == s.State.User.ID {
    return
  }
  debug("Removing ", roleID, " from ", r.UserID)
  err := s.GuildMemberRoleRemove(GuildID, r.UserID, roleID)
  if err != nil {
    log.Print(err)
  }
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
  if m.Author.ID == s.State.User.ID {
    return
  }
  channel, err := s.Channel(m.ChannelID)
  if err != nil {
    log.Print("Error getting channel:")
    log.Print(err)
    return
  }
  guild, err := s.Guild(channel.GuildID)
  if err != nil {
    log.Print("Error getting guild:")
    log.Print(err)
    return
  }
  if m.Author.ID != ownerID && m.Author.ID != guild.OwnerID {
    return
  }
  if strings.HasPrefix(m.Content, "register") {
    if channel.Name != activeChannel {
      debug("register command only works in channels with name: ", activeChannel)
      return
    }
    getRole := regexp.MustCompile(`<@&([0-9]+)>.*\((.*)\)`)
    regexout := getRole.FindAllStringSubmatch(m.Content, -1)
    if regexout != nil && regexout[0][2] != "" {
      roleID := regexout[0][1]
      description := regexout[0][2]
      log.Print("registering ", roleID, ": ", description)
      text := []string{"<@&", roleID, ">\n", description}
      newm, err := s.ChannelMessageSend(m.ChannelID, strings.Join(text, ""))
      if err == nil {
        s.MessageReactionAdd(newm.ChannelID, newm.ID, emoji)
        s.ChannelMessageDelete(channel.ID, m.ID)
      }
    }
  }
}
